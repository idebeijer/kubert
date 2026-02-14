package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/fzf"
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
)

type NamespaceOptions struct {
	Out    io.Writer
	ErrOut io.Writer

	Args []string

	Config            config.Config
	StateManager      func() (*state.Manager, error)
	NamespaceLister   func(ctx context.Context) ([]string, error)
	Selector          func([]string) (string, error)
	IsInteractive     func() bool
	NamespaceSwitcher func(sm *state.Manager, namespace string, namespaces []string) error
}

func NewNamespaceOptions() *NamespaceOptions {
	return &NamespaceOptions{
		Out:    os.Stdout,
		ErrOut: os.Stderr,

		StateManager: state.NewManager,
		NamespaceLister: func(ctx context.Context) ([]string, error) {
			clientset, err := createKubernetesClient()
			if err != nil {
				return nil, err
			}
			return listNamespaces(ctx, clientset)
		},
		Selector:          fzf.Select,
		IsInteractive:     fzf.IsInteractive,
		NamespaceSwitcher: switchNamespace,
	}
}

func NewNamespaceCommand() *cobra.Command {
	o := NewNamespaceOptions()

	cmd := &cobra.Command{
		Use:     "ns",
		Short:   "Switch to a different namespace",
		Long:    `Switch to a different namespace in the current Kubert shell. Other shells with the same context will not be affected.`,
		Aliases: []string{"namespace"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return kubert.ShellPreFlightCheck()
		},
		SilenceUsage:      true,
		ValidArgsFunction: validNamespaceArgsFunction,
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

func (o *NamespaceOptions) Complete(cmd *cobra.Command, args []string) error {
	o.Out = cmd.OutOrStdout()
	o.ErrOut = cmd.ErrOrStderr()
	o.Args = args
	o.Config = config.Cfg
	return nil
}

func (o *NamespaceOptions) Validate() error {
	return nil
}

func (o *NamespaceOptions) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespaces, err := o.NamespaceLister(ctx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timeout listing namespaces: cluster may be unreachable")
		}
		return fmt.Errorf("error listing namespaces: %w", err)
	}

	namespace, err := o.selectNamespace(namespaces)
	if err != nil {
		return err
	}
	if namespace == "" {
		return nil
	}

	sm, err := o.StateManager()
	if err != nil {
		return fmt.Errorf("error creating state manager: %w", err)
	}

	return o.NamespaceSwitcher(sm, namespace, namespaces)
}

func (o *NamespaceOptions) selectNamespace(namespaces []string) (string, error) {
	if len(o.Args) > 0 {
		return o.Args[0], nil
	}

	if !o.IsInteractive() {
		for _, name := range namespaces {
			fmt.Fprintln(o.Out, name)
		}
		return "", nil
	}

	return o.Selector(namespaces)
}

func createKubernetesClient() (*kubernetes.Clientset, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	cfg, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

func listNamespaces(ctx context.Context, clientset kubernetes.Interface) ([]string, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		names = append(names, ns.Name)
	}
	return names, nil
}

func switchNamespace(sm *state.Manager, namespace string, namespaces []string) error {
	if !slices.Contains(namespaces, namespace) {
		return fmt.Errorf("namespace %q does not exist", namespace)
	}

	kubeconfigPath := os.Getenv("KUBECONFIG")
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return err
	}

	if cfg.Contexts == nil {
		return fmt.Errorf("no contexts found in kubeconfig")
	}
	ctx, exists := cfg.Contexts[cfg.CurrentContext]
	if !exists || ctx == nil {
		return fmt.Errorf("current context %q not found in kubeconfig", cfg.CurrentContext)
	}

	ctx.Namespace = namespace

	if err := clientcmd.WriteToFile(*cfg, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	return sm.SetLastNamespaceWithContextCreation(cfg.CurrentContext, namespace)
}

func validNamespaceArgsFunction(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := context.Background()

	clientset, err := createKubernetesClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	namespaces, err := listNamespaces(ctx, clientset)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return namespaces, cobra.ShellCompDirectiveNoFileComp
}
