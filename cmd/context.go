package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sort"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/fzf"
	"github.com/idebeijer/kubert/internal/kubeconfig"
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
)

type ContextOptions struct {
	Out    io.Writer
	ErrOut io.Writer

	Args []string

	Config         config.Config
	ContextLoader  func() ([]kubeconfig.Context, error)
	StateManager   func() (*state.Manager, error)
	Selector       func([]string) (string, error)
	IsInteractive  func() bool
	ShellLauncher  func(kubeconfigPath, originalPath, contextName string, cfg config.Config) error
	TempFileWriter func(kubeconfigPath, contextName, namespace string) (*os.File, func(), error)
}

func NewContextOptions() *ContextOptions {
	return &ContextOptions{
		Out:    os.Stdout,
		ErrOut: os.Stderr,

		ContextLoader: func() ([]kubeconfig.Context, error) {
			cfg := config.Cfg
			fsProvider := kubeconfig.NewFileSystemProvider(cfg.KubeconfigPaths.Include, cfg.KubeconfigPaths.Exclude)
			loader := kubeconfig.NewLoader(kubeconfig.WithProvider(fsProvider))
			return loader.LoadContexts()
		},
		StateManager:  state.NewManager,
		Selector:      fzf.Select,
		IsInteractive: fzf.IsInteractive,
		ShellLauncher: func(kubeconfigPath, originalPath, contextName string, cfg config.Config) error {
			return launchShellWithKubeconfig(kubeconfigPath, originalPath, contextName, cfg)
		},
		TempFileWriter: createTempKubeconfigFile,
	}
}

func NewContextCommand() *cobra.Command {
	o := NewContextOptions()

	cmd := &cobra.Command{
		Use:   "ctx [context-name | -]",
		Short: "Spawn a shell with the selected context",
		Long: `Start a shell with the KUBECONFIG environment variable set to the selected context.
Kubert will issue a temporary kubeconfig file with the selected context, so that multiple shells can be spawned with different contexts.

Use '-' to switch to the previously selected context.`,
		Example: `  # Select a context interactively
  kubert ctx

  # Switch to a specific context
  kubert ctx my-cluster

  # Switch to the previously selected context
  kubert ctx -`,
		Aliases:           []string{"context"},
		SilenceUsage:      true,
		ValidArgsFunction: validContextArgsFunction,
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

	return cmd
}

func (o *ContextOptions) Complete(cmd *cobra.Command, args []string) error {
	o.Out = cmd.OutOrStdout()
	o.ErrOut = cmd.ErrOrStderr()
	o.Args = args
	o.Config = config.Cfg
	return nil
}

func (o *ContextOptions) Validate() error {
	return nil
}

func (o *ContextOptions) Run() error {
	sm, err := o.StateManager()
	if err != nil {
		return fmt.Errorf("error creating state manager: %w", err)
	}

	contexts, err := o.ContextLoader()
	if err != nil {
		return fmt.Errorf("error loading contexts: %w", err)
	}
	slog.Debug("Contexts loaded", "count", len(contexts))

	contextNames := getContextNames(contexts)
	sort.Strings(contextNames)

	selectedContextName, err := o.selectContextName(contextNames, sm)
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

	contextInState, _ := sm.ContextInfo(selectedContextName)
	tempKubeconfig, cleanup, err := o.TempFileWriter(selectedContext.FilePath, selectedContextName, contextInState.LastNamespace)
	if err != nil {
		return err
	}
	defer cleanup()

	slog.Debug("Created a new kubeconfig with the specified context", "tempKubeconfig", tempKubeconfig.Name())

	if err := sm.SetLastContext(selectedContextName); err != nil {
		slog.Warn("Failed to save last context", "error", err)
	}

	return o.ShellLauncher(tempKubeconfig.Name(), selectedContext.FilePath, selectedContextName, o.Config)
}

func (o *ContextOptions) selectContextName(contextNames []string, sm *state.Manager) (string, error) {
	if len(o.Args) > 0 {
		if o.Args[0] != "-" {
			return o.Args[0], nil
		}

		lastContext, exists := sm.GetLastContext()
		if !exists {
			return "", fmt.Errorf("no previous context found")
		}
		return lastContext, nil
	}

	if !o.IsInteractive() {
		o.printContextNames(contextNames)
		return "", nil
	}

	return o.Selector(contextNames)
}

func (o *ContextOptions) printContextNames(contextNames []string) {
	for _, name := range contextNames {
		fmt.Fprintln(o.Out, name)
	}
}

func getContextNames(contexts []kubeconfig.Context) []string {
	names := make([]string, 0, len(contexts))
	for _, context := range contexts {
		names = append(names, context.Name)
	}
	return names
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

func getUserShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh" // Default to /bin/sh if SHELL is not set
	}
	return shell
}

type ShellOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func DefaultShellOptions() ShellOptions {
	return ShellOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func launchShellWithKubeconfig(kubeconfigPath, originalKubeconfigPath, contextName string, cfg config.Config, opts ...ShellOptions) error {
	opt := DefaultShellOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	env := os.Environ()
	env = append(env, "KUBECONFIG="+kubeconfigPath)
	env = append(env, kubert.ShellActiveEnvVar+"=1")
	env = append(env, kubert.ShellKubeconfigEnvVar+"="+kubeconfigPath)
	env = append(env, kubert.ShellOriginalKubeconfigEnvVar+"="+originalKubeconfigPath)
	env = append(env, kubert.ShellContextEnvVar+"="+contextName)

	statefile, _ := state.FilePath()
	env = append(env, kubert.ShellStateFilePathEnvVar+"="+statefile)

	// Execute pre-shell hook if configured
	if cfg.Hooks.PreShell != "" {
		if err := executeHook(cfg.Hooks.PreShell, "pre-shell"); err != nil {
			slog.Warn("Failed to execute pre-shell hook", "error", err)
		}
	}

	// Launch the shell with the current environment, including the modified KUBECONFIG
	shellCmd := exec.Command(getUserShell())
	shellCmd.Env = env
	shellCmd.Stdin = opt.Stdin
	shellCmd.Stdout = opt.Stdout
	shellCmd.Stderr = opt.Stderr

	shellErr := shellCmd.Run()

	// Execute post-shell hook if configured (always run, even if shell exited with error)
	if cfg.Hooks.PostShell != "" {
		if err := executeHook(cfg.Hooks.PostShell, "post-shell"); err != nil {
			slog.Warn("Failed to execute post-shell hook", "error", err)
		}
	}

	if shellErr != nil {
		if exitErr, ok := shellErr.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil // Exit code 130 means the user exited the shell with Ctrl+D, so we don't return an error
		}
		return fmt.Errorf("failed to launch shell: %w", shellErr)
	}

	return nil
}

func executeHook(hookCommand, hookType string) error {
	hookCmd := exec.Command(getUserShell(), "-c", hookCommand)
	hookCmd.Env = os.Environ()
	hookCmd.Stdout = os.Stdout
	hookCmd.Stderr = os.Stderr

	if err := hookCmd.Run(); err != nil {
		return fmt.Errorf("%s hook failed: %w", hookType, err)
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
