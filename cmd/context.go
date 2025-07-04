package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sort"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/fzf"
	"github.com/idebeijer/kubert/internal/kubeconfig"
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func NewContextCommand() *cobra.Command {
	cmd := &cobra.Command{}

	cmd = &cobra.Command{
		Use:   "ctx",
		Short: "Spawn a shell with the selected context",
		Long: `Start a shell with the KUBECONFIG environment variable set to the selected context.
Kubert will issue a temporary kubeconfig file with the selected context, so that multiple shells can be spawned with different contexts.`,
		Aliases:           []string{"context"},
		SilenceUsage:      true,
		ValidArgsFunction: validContextArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Cfg

			fsProvider := kubeconfig.NewFileSystemProvider(cfg.KubeconfigPaths.Include, cfg.KubeconfigPaths.Exclude)
			loader := kubeconfig.NewLoader(
				kubeconfig.WithProvider(fsProvider),
			)

			sm, err := state.NewManager()
			if err != nil {
				return fmt.Errorf("error creating state manager: %w", err)
			}

			contexts, err := loader.LoadContexts()
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
			if selectedContextName == "" {
				return nil
			}

			selectedContext, found := findContextByName(contexts, selectedContextName)
			if !found {
				return fmt.Errorf("context %s not found", selectedContextName)
			}

			// Find the context in the state to get the last namespace used, so it can be set in the new kubeconfig
			contextInState, _ := sm.ContextInfo(selectedContextName)
			tempKubeconfig, cleanup, err := createTempKubeconfigFile(selectedContext.FilePath, selectedContextName, contextInState.LastNamespace)
			if err != nil {
				return err
			}
			defer cleanup()

			slog.Debug("Created a new kubeconfig with the specified context", "tempKubeconfig", tempKubeconfig.Name())

			return launchShellWithKubeconfig(tempKubeconfig.Name(), selectedContext.FilePath)
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

func createTempKubeconfigFile(kubeconfigPath, selectedContextName, namespace string) (*os.File, func(), error) {
	// Load the original kubeconfig
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, nil, err
	}

	selectedContext := cfg.Contexts[selectedContextName]
	if selectedContext == nil {
		return nil, nil, fmt.Errorf("context %s not found in kubeconfig", selectedContextName)
	}
	selectedCluster := cfg.Clusters[selectedContext.Cluster]
	if selectedCluster == nil {
		return nil, nil, fmt.Errorf("cluster %s not found in kubeconfig", selectedContext.Cluster)
	}
	selectedAuthInfo := cfg.AuthInfos[selectedContext.AuthInfo]
	if selectedAuthInfo == nil {
		return nil, nil, fmt.Errorf("auth info %s not found in kubeconfig", selectedContext.AuthInfo)
	}

	// Build a new kubeconfig with only the selected context
	newConfig := api.NewConfig()
	newConfig.Contexts[selectedContextName] = selectedContext
	newConfig.Clusters[selectedContext.Cluster] = selectedCluster
	newConfig.AuthInfos[selectedContext.AuthInfo] = selectedAuthInfo
	newConfig.CurrentContext = selectedContextName
	if namespace != "" {
		newConfig.Contexts[selectedContextName].Namespace = namespace
	}

	tempKubeconfig, err := os.CreateTemp("", "kubert-*.yaml")
	if err != nil {
		return nil, nil, err
	}
	err = os.Chmod(tempKubeconfig.Name(), 0o600)
	if err != nil {
		return nil, nil, err
	}

	if err := clientcmd.WriteToFile(*newConfig, tempKubeconfig.Name()); err != nil {
		return nil, nil, fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	cleanup := func() {
		_ = tempKubeconfig.Close()
		_ = os.Remove(tempKubeconfig.Name())
	}

	return tempKubeconfig, cleanup, nil
}

func launchShellWithKubeconfig(kubeconfigPath, originalKubeconfigPath string) error {
	// Set the KUBECONFIG environment variable to the path of the temporary kubeconfig file
	if err := os.Setenv("KUBECONFIG", kubeconfigPath); err != nil {
		return fmt.Errorf("failed to set KUBECONFIG environment variable: %w", err)
	}
	if err := os.Setenv(kubert.ShellActiveEnvVar, "1"); err != nil {
		return fmt.Errorf("failed to set KUBERT_SHELL environment variable: %w", err)
	}
	if err := os.Setenv(kubert.ShellKubeconfigEnvVar, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to set KUBERT_SHELL_KUBECONFIG environment variable: %w", err)
	}
	if err := os.Setenv(kubert.ShellOriginalKubeconfigEnvVar, originalKubeconfigPath); err != nil {
		return fmt.Errorf("failed to set KUBERT_SHELL_ORIGINAL_KUBECONFIG environment variable: %w", err)
	}

	statefile, _ := state.FilePath()
	if err := os.Setenv(kubert.ShellStateFilePathEnvVar, statefile); err != nil {
		return fmt.Errorf("failed to set KUBERT_SHELL_STATE_FILE environment variable: %w", err)
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
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil // Exit code 130 means the user exited the shell with Ctrl+D, so we don't return an error
		}
		return fmt.Errorf("failed to launch shell: %w", err)
	}

	return nil
}

func validContextArgsFunction(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg := config.Cfg
	fsProvider := kubeconfig.NewFileSystemProvider(cfg.KubeconfigPaths.Include, cfg.KubeconfigPaths.Exclude)
	loader := kubeconfig.NewLoader(kubeconfig.WithProvider(fsProvider))

	contexts, err := loader.LoadContexts()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	contextNames := getContextNames(contexts)
	sort.Strings(contextNames)

	return contextNames, cobra.ShellCompDirectiveNoFileComp
}
