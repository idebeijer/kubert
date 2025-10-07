package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/fzf"
	"github.com/idebeijer/kubert/internal/kubeconfig"
	"github.com/idebeijer/kubert/internal/state"
)

type execFlags struct {
	contexts  string
	namespace string
	regex     bool
	parallel  bool
	dryRun    bool
}

type contextExecResult struct {
	contextName string
	output      string
	err         error
}

func NewExecCommand() *cobra.Command {
	flags := &execFlags{}

	cmd := &cobra.Command{
		Use:   "exec [flags] -- command [args...]",
		Short: "Execute a command against multiple contexts",
		Long: `Execute a command against multiple Kubernetes contexts matching a pattern.

The command will run against all contexts matching the provided pattern.
By default, uses glob-style wildcards (* and ?). Use --regex for regex patterns.

If --contexts is not provided and running in an interactive shell with fzf,
you can select multiple contexts interactively (use Tab/Shift-Tab to select).`,
		Example: `  # Run kubectl get pods in all production contexts
  kubert exec --contexts "prod*" -- kubectl get pods

  # Use regex to match specific patterns
  kubert exec --contexts "^(test|staging).*" --regex -- kubectl get nodes

  # Run in parallel across contexts
  kubert exec --contexts "staging*" --parallel -- kubectl get deployments

  # Specify namespace for all contexts
  kubert exec --contexts "prod*" --namespace kube-system -- kubectl get pods
  
  # Interactive multi-select (if fzf is available)
  kubert exec -- kubectl get nodes
  
  # Dry run to see which contexts will be used
  kubert exec --contexts "prod*" --dry-run -- kubectl get pods`,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Cfg
			fsProvider := kubeconfig.NewFileSystemProvider(cfg.KubeconfigPaths.Include, cfg.KubeconfigPaths.Exclude)
			loader := kubeconfig.NewLoader(kubeconfig.WithProvider(fsProvider))

			contexts, err := loader.LoadContexts()
			if err != nil {
				return fmt.Errorf("error loading contexts: %w", err)
			}

			var matchedContexts []kubeconfig.Context

			if flags.contexts == "" {
				if !fzf.IsInteractiveShell() {
					return fmt.Errorf("--contexts flag is required in non-interactive mode")
				}

				contextNames := getContextNames(contexts)
				sort.Strings(contextNames)

				selectedNames, err := fzf.SelectMulti(contextNames)
				if err != nil {
					return fmt.Errorf("context selection cancelled or failed: %w", err)
				}

				if len(selectedNames) == 0 {
					return fmt.Errorf("no contexts selected")
				}

				for _, name := range selectedNames {
					ctx, found := findContextByName(contexts, name)
					if found {
						matchedContexts = append(matchedContexts, ctx)
					}
				}
			} else {
				matchedContexts, err = filterContextsByPattern(contexts, flags.contexts, flags.regex)
				if err != nil {
					return fmt.Errorf("error filtering contexts: %w", err)
				}

				if len(matchedContexts) == 0 {
					return fmt.Errorf("no contexts matched the pattern: %s", flags.contexts)
				}
			}

			sm, err := state.NewManager()
			if err != nil {
				return fmt.Errorf("error creating state manager: %w", err)
			}

			if flags.dryRun {
				return showDryRun(matchedContexts, args, flags.namespace, sm, cfg)
			}

			fmt.Printf("Executing command against %d context(s):\n", len(matchedContexts))
			for _, ctx := range matchedContexts {
				fmt.Printf("  - %s\n", ctx.Name)
			}
			fmt.Println()

			if flags.parallel {
				return executeParallel(matchedContexts, args, flags.namespace, sm, cfg)
			}
			return executeSequential(matchedContexts, args, flags.namespace, sm, cfg)
		},
	}

	cmd.Flags().StringVarP(&flags.contexts, "contexts", "c", "", "Pattern to match context names (omit for interactive multi-select)")
	cmd.Flags().StringVarP(&flags.namespace, "namespace", "n", "default", "Namespace to use for all contexts")
	cmd.Flags().BoolVar(&flags.regex, "regex", false, "Use regex pattern matching instead of glob-style wildcards")
	cmd.Flags().BoolVarP(&flags.parallel, "parallel", "p", false, "Execute commands in parallel across all contexts")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Show which contexts would be used without executing the command")

	return cmd
}

func filterContextsByPattern(contexts []kubeconfig.Context, pattern string, useRegex bool) ([]kubeconfig.Context, error) {
	var regexPattern string

	if useRegex {
		regexPattern = pattern
	} else {
		regexPattern = globToRegex(pattern)
	}

	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var matched []kubeconfig.Context
	for _, ctx := range contexts {
		if regex.MatchString(ctx.Name) {
			matched = append(matched, ctx)
		}
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Name < matched[j].Name
	})

	return matched, nil
}

func globToRegex(pattern string) string {
	pattern = regexp.QuoteMeta(pattern)
	pattern = strings.ReplaceAll(pattern, `\*`, ".*")
	pattern = strings.ReplaceAll(pattern, `\?`, ".")
	return "^" + pattern + "$"
}

func executeSequential(contexts []kubeconfig.Context, args []string, namespace string, sm *state.Manager, cfg config.Config) error {
	hasErrors := false

	for i, ctx := range contexts {
		if i > 0 {
			fmt.Println()
		}

		result := executeInContext(ctx, args, namespace, sm, cfg)
		printResult(result)

		if result.err != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("one or more commands failed")
	}

	return nil
}

func executeParallel(contexts []kubeconfig.Context, args []string, namespace string, sm *state.Manager, cfg config.Config) error {
	var wg sync.WaitGroup
	resultsChan := make(chan contextExecResult, len(contexts))

	for _, ctx := range contexts {
		wg.Add(1)
		go func(ctx kubeconfig.Context) {
			defer wg.Done()
			result := executeInContext(ctx, args, namespace, sm, cfg)
			resultsChan <- result
		}(ctx)
	}

	wg.Wait()
	close(resultsChan)

	var results []contextExecResult
	for result := range resultsChan {
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].contextName < results[j].contextName
	})

	hasErrors := false
	for i, result := range results {
		if i > 0 {
			fmt.Println()
		}
		printResult(result)
		if result.err != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("one or more commands failed")
	}

	return nil
}

func executeInContext(ctx kubeconfig.Context, args []string, namespace string, sm *state.Manager, cfg config.Config) contextExecResult {
	result := contextExecResult{
		contextName: ctx.Name,
	}

	locked, err := isContextProtected(sm, ctx.Name, cfg)
	if err != nil {
		result.err = fmt.Errorf("error checking context protection: %w", err)
		return result
	}

	if locked {
		if cfg.Contexts.ExitOnProtectedKubectlCmd {
			result.err = fmt.Errorf("context is protected and ExitOnProtectedKubectlCmd is enabled")
			return result
		}

		yellow := color.New(color.FgHiYellow).SprintFunc()
		result.output = fmt.Sprintf("%s: context %s is protected, skipping...\n", yellow("WARNING"), ctx.Name)
		return result
	}

	tempKubeconfig, cleanup, err := createTempKubeconfigFile(ctx.FilePath, ctx.Name, namespace)
	if err != nil {
		result.err = fmt.Errorf("failed to create temp kubeconfig: %w", err)
		return result
	}
	defer cleanup()

	output, err := runCommand(args, tempKubeconfig.Name())
	result.output = output
	result.err = err

	return result
}

func runCommand(args []string, kubeconfigPath string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func printResult(result contextExecResult) {
	separator := strings.Repeat("=", 80)
	contextHeader := fmt.Sprintf("Context: %s", result.contextName)

	fmt.Println(separator)
	fmt.Println(contextHeader)
	fmt.Println(separator)

	if result.err != nil {
		red := color.New(color.FgRed).SprintFunc()
		fmt.Printf("%s: %v\n", red("ERROR"), result.err)
		if result.output != "" {
			fmt.Println(result.output)
		}
	} else {
		fmt.Print(result.output)
	}
}

func showDryRun(contexts []kubeconfig.Context, args []string, namespace string, sm *state.Manager, cfg config.Config) error {
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Println("=== DRY RUN ===")
	fmt.Println()
	fmt.Printf("Command: %s\n", strings.Join(args, " "))
	if namespace != "" {
		fmt.Printf("Namespace: %s\n", namespace)
	}
	fmt.Printf("Total contexts: %d\n", len(contexts))
	fmt.Println()

	fmt.Println("Contexts to execute against:")
	for _, ctx := range contexts {
		locked, err := isContextProtected(sm, ctx.Name, cfg)
		if err != nil {
			return fmt.Errorf("error checking context protection for %s: %w", ctx.Name, err)
		}

		status := green("✓")
		statusText := ""
		if locked {
			if cfg.Contexts.ExitOnProtectedKubectlCmd {
				status = yellow("⊘")
				statusText = " (protected - will be skipped)"
			} else {
				status = yellow("⚠")
				statusText = " (protected - will prompt)"
			}
		}

		fmt.Printf("  %s %s%s\n", status, ctx.Name, statusText)
	}

	return nil
}
