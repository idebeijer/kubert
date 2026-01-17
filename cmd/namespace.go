package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/idebeijer/kubert/internal/fzf"
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
)

func NewNamespaceCommand() *cobra.Command {
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
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			sm, err := state.NewManager()
			if err != nil {
				return err
			}

			clientset, err := createKubernetesClient()
			if err != nil {
				return err
			}

			namespaces, err := listNamespaces(ctx, clientset)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					return fmt.Errorf("timeout listing namespaces: cluster may be unreachable")
				}
				return err
			}

			namespace, err := selectNamespace(args, namespaces)
			if err != nil {
				return err
			}

			if err := switchNamespace(sm, namespace, namespaces); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// createKubernetesClient creates a Kubernetes client from the kubeconfig
func createKubernetesClient() (*kubernetes.Clientset, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// listNamespaces lists all namespaces in the Kubernetes cluster
func listNamespaces(ctx context.Context, clientset *kubernetes.Clientset) ([]string, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var namespaceNames []string
	for _, ns := range namespaces.Items {
		namespaceNames = append(namespaceNames, ns.Name)
	}
	return namespaceNames, nil
}

func selectNamespace(args []string, namespaces []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	if !fzf.IsInteractiveShell() {
		printNamespaces(namespaces)
		return "", nil
	}
	return fzf.Select(namespaces)
}

func printNamespaces(contextNames []string) {
	for _, name := range contextNames {
		fmt.Println(name)
	}
}

func switchNamespace(sm *state.Manager, namespace string, namespaces []string) error {
	namespaceExists := false
	for _, ns := range namespaces {
		if ns == namespace {
			namespaceExists = true
			break
		}
	}
	if !namespaceExists {
		return fmt.Errorf("namespace \"%s\" does not exist", namespace)
	}

	kubeconfigPath := os.Getenv("KUBECONFIG")
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return err
	}

	config.Contexts[config.CurrentContext].Namespace = namespace

	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	if err := sm.SetLastNamespaceWithContextCreation(config.CurrentContext, namespace); err != nil {
		return err
	}

	return nil
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
