package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
	"github.com/spf13/cobra"
)

func NewKubectlCommand() *cobra.Command {
	cmd := &cobra.Command{}

	cmd = &cobra.Command{
		Use:                "kubectl",
		Short:              "Wrapper for kubectl",
		Long:               `Wrapper for kubectl, to support context locking.`,
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

			locked, err := isContextLocked(sm, clientConfig.CurrentContext, cfg)
			if err != nil {
				return err
			}

			if locked && isCommandBlocked(args, cfg.Contexts.BlockedKubectlCommands) {
				fmt.Printf("Oops: you tried to run the kubectl command \"%s\" in the locked context \"%s\".\n\n"+
					"The command has not been executed because the \"%s\" command is on the blocked list, and the current context is locked.\n"+
					"Use 'kubert context-lock unlock' to unlock the current context.\n"+
					"Exiting...\n", args[0], clientConfig.CurrentContext, args[0])
				return nil
			}

			kubectlCmd := exec.Command("kubectl", args...)
			kubectlCmd.Stdin = os.Stdin
			kubectlCmd.Stdout = os.Stdout
			kubectlCmd.Stderr = os.Stderr
			return kubectlCmd.Run()
		},
	}

	return cmd
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

func isCommandBlocked(args []string, blockedCmds []string) bool {
	if len(args) > 0 {
		for _, blockedCmd := range blockedCmds {
			if args[0] == blockedCmd {
				return true
			}
		}
	}
	return false
}

func isContextLocked(sm *state.Manager, context string, cfg config.Config) (bool, error) {
	contextInfo, _ := sm.ContextInfo(context)
	if contextInfo.Locked == nil && cfg.Contexts.DefaultLocked != nil {
		regex, err := regexp.Compile(*cfg.Contexts.DefaultLocked)
		if err != nil {
			return false, fmt.Errorf("failed to compile regex: %w", err)
		}

		if regex.MatchString(context) {
			return true, nil
		}
	}

	if contextInfo.Locked != nil && *contextInfo.Locked == true {
		return true, nil
	}

	return false, nil
}
