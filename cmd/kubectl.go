package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func NewKubectlCommand() *cobra.Command {
	cmd := &cobra.Command{}

	cmd = &cobra.Command{
		Use:                "kubectl",
		Short:              "Wrapper for kubectl, with some extra features",
		Long:               `Wrapper for kubectl, with some extra features`,
		DisableFlagParsing: true,
		Aliases:            []string{"namespace"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return kubectlPreflightCheck()
		},
		ValidArgsFunction: validKubectlArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
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

func kubectlPreflightCheck() error {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return fmt.Errorf("kubectl not found in PATH")
	}
	return nil
}
