package cmd

import (
	"encoding/json"
	"fmt"
	"io"
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

const outputJSON = "json"

var execExample = `  # Run kubectl get pods in all production contexts
  kubert exec "prod*" -- kubectl get pods

  # Match multiple patterns
  kubert exec "prod*" "staging*" -- kubectl get nodes

  # Use regex to match specific patterns
  kubert exec --regex "^(test|staging).*" -- kubectl get nodes

  # Run in parallel across contexts
  kubert exec "staging*" --parallel -- kubectl get deployments
  
  # Aggregate structured output across contexts into JSON array
  kubert exec "prod*" -o json -- kubectl get nodes -o json | jq '.[].output.items[]?'

  # Interactive multi-select (if fzf is available)
  kubert exec -- kubectl get nodes
  
  # Dry run to see which contexts would be used
  kubert exec "prod*" --dry-run -- kubectl get pods
`

type ExecOptions struct {
	Out    io.Writer
	ErrOut io.Writer

	Namespace string
	Regex     bool
	Parallel  bool
	DryRun    bool
	Output    string

	Patterns    []string
	CommandArgs []string

	Config        config.Config
	ContextLoader func() ([]kubeconfig.Context, error)
	StateManager  func() (*state.Manager, error)
	IsInteractive func() bool
	Selector      func([]string) ([]string, error)
}

func NewExecOptions() *ExecOptions {
	return &ExecOptions{
		Out:    os.Stdout,
		ErrOut: os.Stderr,

		ContextLoader: func() ([]kubeconfig.Context, error) {
			cfg := config.Cfg
			fsProvider := kubeconfig.NewFileSystemProvider(cfg.KubeconfigPaths.Include, cfg.KubeconfigPaths.Exclude)
			loader := kubeconfig.NewLoader(kubeconfig.WithProvider(fsProvider))
			return loader.LoadContexts()
		},
		StateManager:  state.NewManager,
		IsInteractive: fzf.IsInteractive,
		Selector:      fzf.SelectMulti,
	}
}

func NewExecCommand() *cobra.Command {
	o := NewExecOptions()

	cmd := &cobra.Command{
		Use:   "exec [pattern...] -- command [args...]",
		Short: "Execute a command against multiple contexts",
		Long: `Execute a command against multiple Kubernetes contexts matching one or more patterns.

The command will run against all contexts matching the provided patterns.
By default, uses glob-style wildcards (* and ?). Use --regex for regex patterns.

If no patterns are provided and running in an interactive shell with fzf,
you can select multiple contexts interactively (use Tab/Shift-Tab to select).`,
		Example:      execExample,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(cmd, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "Namespace to use for all contexts (defaults to each context's configured namespace)")
	cmd.Flags().BoolVar(&o.Regex, "regex", false, "Use regex pattern matching instead of glob-style wildcards")
	cmd.Flags().BoolVarP(&o.Parallel, "parallel", "p", false, "Execute commands in parallel across all contexts")
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "Show which contexts would be used without executing the command")
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "Output format (e.g., 'json')")

	return cmd
}

func (o *ExecOptions) Complete(cmd *cobra.Command, args []string) error {
	o.Out = cmd.OutOrStdout()
	o.ErrOut = cmd.ErrOrStderr()
	o.Config = config.Cfg

	dashIdx := cmd.ArgsLenAtDash()
	switch dashIdx {
	case -1:
		return fmt.Errorf("missing '--' separator between patterns and command")
	case 0:
		o.Patterns = []string{}
		o.CommandArgs = args
	default:
		o.Patterns = args[:dashIdx]
		o.CommandArgs = args[dashIdx:]
	}

	return nil
}

func (o *ExecOptions) Validate() error {
	if len(o.CommandArgs) == 0 {
		return fmt.Errorf("no command provided after '--'")
	}

	if len(o.Patterns) == 0 && !o.IsInteractive() {
		return fmt.Errorf("patterns are required in non-interactive mode")
	}

	if o.Output != "" && o.Output != outputJSON {
		return fmt.Errorf("invalid output format: %s. Only '%s' is supported", o.Output, outputJSON)
	}

	return nil
}

func (o *ExecOptions) Run() error {
	contexts, err := o.ContextLoader()
	if err != nil {
		return fmt.Errorf("error loading contexts: %w", err)
	}

	matchedContexts, err := o.resolveContexts(contexts)
	if err != nil {
		return err
	}

	sm, err := o.StateManager()
	if err != nil {
		return fmt.Errorf("error creating state manager: %w", err)
	}

	if o.DryRun {
		return showDryRun(o.Out, matchedContexts, o.CommandArgs, o.Namespace, sm, o.Config)
	}

	if o.Output != outputJSON {
		fmt.Fprintf(o.Out, "Executing command against %d context(s):\n", len(matchedContexts))
		for _, ctx := range matchedContexts {
			fmt.Fprintf(o.Out, "  - %s\n", ctx.Name)
		}
		fmt.Fprintln(o.Out)
	}

	if o.Parallel {
		return executeParallel(o.Out, matchedContexts, o.CommandArgs, o.Namespace, sm, o.Config, o.Output)
	}
	return executeSequential(o.Out, matchedContexts, o.CommandArgs, o.Namespace, sm, o.Config, o.Output)
}

func (o *ExecOptions) resolveContexts(contexts []kubeconfig.Context) ([]kubeconfig.Context, error) {
	if len(o.Patterns) == 0 {
		return o.resolveInteractive(contexts)
	}

	matched, err := filterContextsByPatterns(contexts, o.Patterns, o.Regex)
	if err != nil {
		return nil, fmt.Errorf("error filtering contexts: %w", err)
	}

	if len(matched) == 0 {
		return nil, fmt.Errorf("no contexts matched the patterns: %s", strings.Join(o.Patterns, ", "))
	}

	return matched, nil
}

func (o *ExecOptions) resolveInteractive(contexts []kubeconfig.Context) ([]kubeconfig.Context, error) {
	contextNames := getContextNames(contexts)
	sort.Strings(contextNames)

	selectedNames, err := o.Selector(contextNames)
	if err != nil {
		return nil, fmt.Errorf("context selection cancelled or failed: %w", err)
	}

	if len(selectedNames) == 0 {
		return nil, fmt.Errorf("no contexts selected")
	}

	var matched []kubeconfig.Context
	for _, name := range selectedNames {
		if ctx, found := findContextByName(contexts, name); found {
			matched = append(matched, ctx)
		}
	}

	return matched, nil
}

type contextExecResult struct {
	contextName string
	output      string
	err         error
}

func filterContextsByPatterns(contexts []kubeconfig.Context, patterns []string, useRegex bool) ([]kubeconfig.Context, error) {
	matchedMap := make(map[string]kubeconfig.Context)

	for _, pattern := range patterns {
		matched, err := filterContextsByPattern(contexts, pattern, useRegex)
		if err != nil {
			return nil, err
		}

		for _, ctx := range matched {
			matchedMap[ctx.Name] = ctx
		}
	}

	var result []kubeconfig.Context
	for _, ctx := range matchedMap {
		result = append(result, ctx)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
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

func executeSequential(out io.Writer, contexts []kubeconfig.Context, args []string, namespace string, sm *state.Manager, cfg config.Config, outputFormat string) error {
	hasErrors := false
	var allResults []contextExecResult

	for i, ctx := range contexts {
		if outputFormat != outputJSON && i > 0 {
			fmt.Fprintln(out)
		}

		result := executeInContext(ctx, args, namespace, sm, cfg, outputFormat)

		if outputFormat == outputJSON {
			allResults = append(allResults, result)
		} else {
			printResult(out, result)
		}

		if result.err != nil {
			hasErrors = true
		}
	}

	if outputFormat == outputJSON {
		printJSONResults(out, allResults)
	}

	if hasErrors {
		return fmt.Errorf("one or more commands failed")
	}

	return nil
}

func executeParallel(out io.Writer, contexts []kubeconfig.Context, args []string, namespace string, sm *state.Manager, cfg config.Config, outputFormat string) error {
	var wg sync.WaitGroup

	resultsChan := make(chan contextExecResult, len(contexts))

	for _, ctx := range contexts {
		wg.Add(1)
		go func(ctx kubeconfig.Context) {
			defer wg.Done()
			result := executeInContext(ctx, args, namespace, sm, cfg, outputFormat)
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
		if outputFormat != outputJSON {
			if i > 0 {
				fmt.Fprintln(out)
			}
			printResult(out, result)
		}

		if result.err != nil {
			hasErrors = true
		}
	}

	if outputFormat == outputJSON {
		printJSONResults(out, results)
	}

	if hasErrors {
		return fmt.Errorf("one or more commands failed")
	}

	return nil
}

func executeInContext(ctx kubeconfig.Context, args []string, namespace string, sm *state.Manager, cfg config.Config, outputFormat string) contextExecResult {
	result := contextExecResult{
		contextName: ctx.Name,
	}

	locked, err := isContextProtected(sm, ctx.Name, cfg)
	if err != nil {
		result.err = fmt.Errorf("error checking context protection: %w", err)
		return result
	}

	if locked {
		warningText := "WARNING"
		if outputFormat != outputJSON {
			yellow := color.New(color.FgHiYellow).SprintFunc()
			warningText = yellow(warningText)
		}
		result.err = fmt.Errorf("%s: context %s is protected, skipping", warningText, ctx.Name)
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

func printResult(out io.Writer, result contextExecResult) {
	separator := strings.Repeat("=", 80)
	contextHeader := fmt.Sprintf("Context: %s", result.contextName)

	fmt.Fprintln(out, separator)
	fmt.Fprintln(out, contextHeader)
	fmt.Fprintln(out, separator)

	if result.err != nil {
		red := color.New(color.FgRed).SprintFunc()
		fmt.Fprintf(out, "%s: %v\n", red("ERROR"), result.err)
		if result.output != "" {
			fmt.Fprintln(out, result.output)
		}
	} else {
		fmt.Fprint(out, result.output)
	}
}

func printJSONResults(out io.Writer, results []contextExecResult) {
	outputList := make([]map[string]any, 0, len(results))

	for _, res := range results {
		var jsonObj any
		outputStr := strings.TrimSpace(res.output)

		if outputStr != "" {
			err := json.Unmarshal([]byte(outputStr), &jsonObj)
			if err != nil {
				jsonObj = outputStr
			}
		} else {
			jsonObj = ""
		}

		resMap := make(map[string]any)
		resMap["context"] = res.contextName

		if res.err != nil {
			resMap["error"] = res.err.Error()
		}
		resMap["output"] = jsonObj

		outputList = append(outputList, resMap)
	}

	jsonBytes, err := json.MarshalIndent(outputList, "", "  ")
	if err != nil {
		fallbackError := []map[string]string{
			{"error": fmt.Sprintf("failed to marshal json: %v", err)},
		}
		fallbackBytes, _ := json.MarshalIndent(fallbackError, "", "  ")
		fmt.Fprintln(out, string(fallbackBytes))
		return
	}
	fmt.Fprintln(out, string(jsonBytes))
}

func showDryRun(out io.Writer, contexts []kubeconfig.Context, args []string, namespace string, sm *state.Manager, cfg config.Config) error {
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Fprintln(out, "=== DRY RUN ===")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Command: %s\n", strings.Join(args, " "))
	if namespace != "" {
		fmt.Fprintf(out, "Namespace: %s\n", namespace)
	}
	fmt.Fprintf(out, "Total contexts: %d\n", len(contexts))
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Contexts to execute against:")
	for _, ctx := range contexts {
		locked, err := isContextProtected(sm, ctx.Name, cfg)
		if err != nil {
			return fmt.Errorf("error checking context protection for %s: %w", ctx.Name, err)
		}

		status := green("✓")
		statusText := ""
		if locked {
			if !cfg.Protection.Prompt {
				status = yellow("⊘")
				statusText = " (protected - will be skipped)"
			} else {
				status = yellow("⚠")
				statusText = " (protected - will prompt)"
			}
		}

		fmt.Fprintf(out, "  %s %s%s\n", status, ctx.Name, statusText)
	}

	return nil
}
