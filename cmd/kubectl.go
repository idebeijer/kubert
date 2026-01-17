package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
)

func NewKubectlCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "kubectl",
		Short:              "Wrapper for kubectl",
		Long:               `Wrapper for kubectl, to support context protection with "kubert context-protection".`,
		DisableFlagParsing: true,
		Aliases:            []string{"namespace"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			_, err := exec.LookPath("kubectl")
			if err != nil {
				return fmt.Errorf("kubectl not found in PATH")
			}

			return kubert.ShellPreFlightCheck()
		},
		SilenceUsage:      true,
		ValidArgsFunction: validKubectlArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Cfg

			sm, err := state.NewManager()
			if err != nil {
				return err
			}

			clientConfig, err := util.KubeClientConfig()
			if err != nil {
				return err
			}

			locked, err := isContextProtected(sm, clientConfig.CurrentContext, cfg)
			if err != nil {
				return err
			}

			if locked && isCommandProtected(args, cfg.Contexts.ProtectedKubectlCommands) {

				if cfg.Contexts.ExitOnProtectedKubectlCmd {
					fmt.Printf("You tried to run the protected kubectl command \"%s\" in the protected context \"%s\".\n\n"+
						"The command has not been executed and kubert will exit immediately.\n"+
						"Exiting...\n", args[0], clientConfig.CurrentContext)
					return nil
				}

				yellow := color.New(color.FgHiYellow).SprintFunc()
				fmt.Printf("%s: you tried to run the protected kubectl command \"%s\" in the protected context \"%s\".\n\n", yellow("WARNING"), args[0], clientConfig.CurrentContext)
				if !promptUserConfirmation() {
					fmt.Println("Exiting...")
					return nil
				}
				fmt.Println()
			}

			kubectlCmd := exec.Command("kubectl", args...)
			kubectlCmd.Stdin = os.Stdin
			kubectlCmd.Stdout = os.Stdout
			kubectlCmd.Stderr = os.Stderr
			if err := kubectlCmd.Run(); err != nil {
				if _, ok := err.(*exec.ExitError); ok {
					// Return nil to avoid duplicating the error message given by kubectl
					return nil
				}
				return fmt.Errorf("kubectl error: %w", err)
			}
			return nil
		},
	}

	return cmd
}

func promptUserConfirmation() bool {
	var response string
	fmt.Print("Are you sure you want to continue? [y/N]: ")
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

func validKubectlArgsFunction(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Prepare kubectl completion command
	compCmd := append([]string{"__complete"}, args...)
	compCmd = append(compCmd, toComplete)

	// Set up environment variables for kubectl completion
	env := os.Environ()
	env = append(env, "COMP_LINE=kubectl "+strings.Join(append(args, toComplete), " "))
	env = append(env, fmt.Sprintf("COMP_POINT=%d", len("kubectl ")+len(strings.Join(append(args, toComplete), " "))))

	kubectlComp := exec.Command("kubectl", compCmd...)
	kubectlComp.Env = env
	out, err := kubectlComp.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "completion error: %v", err)
		return nil, cobra.ShellCompDirectiveDefault
	}

	completions := strings.Split(strings.TrimSpace(string(out)), "\n")
	var validCompletions []string
	for _, completion := range completions {
		// Filter out invalid or unexpected results // TODO: improve this, might be a bit hacky. Completion would show ":4" for example
		if strings.HasPrefix(completion, ":") {
			continue
		}
		validCompletions = append(validCompletions, completion)
	}
	return validCompletions, cobra.ShellCompDirectiveDefault
}

func isCommandProtected(args []string, blockedCmds []string) bool {
	if len(args) > 0 {
		if slices.Contains(blockedCmds, args[0]) {
			return true
		}
	}
	return false
}

func isContextProtected(sm *state.Manager, context string, cfg config.Config) (bool, error) {
	contextInfo, _ := sm.ContextInfo(context)
	if contextInfo.Protected == nil && cfg.Contexts.ProtectedByDefaultRegexp != nil {
		regex, err := regexp.Compile(*cfg.Contexts.ProtectedByDefaultRegexp)
		if err != nil {
			return false, fmt.Errorf("failed to compile regex: %w", err)
		}

		if regex.MatchString(context) {
			return true, nil
		}
	}

	if contextInfo.Protected != nil && *contextInfo.Protected {
		return true, nil
	}

	return false, nil
}
