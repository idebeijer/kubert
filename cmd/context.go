package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/fzf"
	"github.com/idebeijer/kubert/internal/kubeconfig"
	"github.com/spf13/cobra"
)

func NewContextCommand() *cobra.Command {
	cmd := &cobra.Command{}

	cmd = &cobra.Command{
		Use:     "ctx",
		Short:   "context command",
		Aliases: []string{"context"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Cfg
			fsProvider := kubeconfig.NewFileSystemProvider(cfg.KubeconfigPaths.Include, cfg.KubeconfigPaths.Exclude)
			loader := kubeconfig.NewLoader(fsProvider)
			contextLoader := kubeconfig.NewContextLoader(loader)

			contexts, err := contextLoader.LoadContexts()
			if err != nil {
				return fmt.Errorf("error loading contexts: %w", err)
			}
			slog.Debug("Contexts loaded", "count", len(contexts))

			contextNames := getContextNames(contexts)
			sort.Strings(contextNames)

			selectedContextName, err := selectContextName(args, contextNames)
			if err != nil {
				return err
			}

			selectedContext, found := findContextByName(contexts, selectedContextName)
			if !found {
				return fmt.Errorf("context %s not found", selectedContextName)
			}

			tempKubeconfig, cleanup, err := createTempKubeconfigFile(selectedContext.FilePath)
			if err != nil {
				return err
			}
			defer cleanup()

			slog.Debug("Found and copied the specified kubeconfig to a temp file", "tempKubeconfig", tempKubeconfig.Name())

			return launchShellWithKubeconfig(tempKubeconfig.Name())
		},
	}

	return cmd
}

func getContextNames(contexts []kubeconfig.Context) []string {
	var names []string
	for _, context := range contexts {
		names = append(names, context.Name)
	}
	return names
}

func selectContextName(args []string, contextNames []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	if !fzf.IsInteractiveShell() {
		printContextNames(contextNames)
		return "", nil
	}
	return fzf.Select(contextNames)
}

func printContextNames(contextNames []string) {
	for _, name := range contextNames {
		fmt.Println(name)
	}
}

func findContextByName(contexts []kubeconfig.Context, name string) (kubeconfig.Context, bool) {
	for _, context := range contexts {
		if context.Name == name {
			return context, true
		}
	}
	return kubeconfig.Context{}, false
}

func createTempKubeconfigFile(kubeconfigPath string) (*os.File, func(), error) {
	tempKubeconfig, err := os.CreateTemp("", "kubert-*.yaml")
	if err != nil {
		return nil, nil, err
	}
	err = os.Chmod(tempKubeconfig.Name(), 0600)
	if err != nil {
		return nil, nil, err
	}
	selectedKubeconfig, err := os.Open(strings.TrimSpace(kubeconfigPath))
	if err != nil {
		return nil, nil, err
	}
	_, err = io.Copy(tempKubeconfig, selectedKubeconfig)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = tempKubeconfig.Close()
		_ = selectedKubeconfig.Close()
		_ = os.Remove(tempKubeconfig.Name())
	}

	return tempKubeconfig, cleanup, nil
}

func launchShellWithKubeconfig(kubeconfigPath string) error {
	// Set the KUBECONFIG environment variable to the path of the temporary kubeconfig file
	if err := os.Setenv("KUBECONFIG", kubeconfigPath); err != nil {
		return fmt.Errorf("failed to set KUBECONFIG environment variable: %w", err)
	}

	// Get the user's preferred shell from the SHELL environment variable
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh" // Default to /bin/sh if SHELL is not set
	}

	// Launch the shell with the current environment, including the modified KUBECONFIG
	shellCmd := exec.Command(shell)
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	if err := shellCmd.Run(); err != nil {
		return fmt.Errorf("failed to launch shell: %w", err)
	}

	return nil
}
