package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/idebeijer/kubert/internal/fzf"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewNamespaceCommand() *cobra.Command {
	cmd := &cobra.Command{}

	cmd = &cobra.Command{
		Use:     "ns",
		Short:   "Switch to a different namespace",
		Long:    `Switch to a different namespace in the current Kubert shell. Other shells with the same context will not be affected.`,
		Aliases: []string{"namespace"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return preflightCheck()
		},
		ValidArgsFunction: validNamespaceArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

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
	if kubeconfigPath == "" {
		kubeconfigPath = clientcmd.RecommendedHomeFile
	}

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

func preflightCheck() error {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = clientcmd.RecommendedHomeFile
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig file not found at %s", kubeconfigPath)
	}

	if kubertActive := os.Getenv(KubertShellActiveEnvVar); kubertActive != "1" {
		return fmt.Errorf("shell not started by kubert")
	}

	kubertKubeconfig := os.Getenv(KubertShellKubeconfigEnvVar)
	if kubertKubeconfig == "" {
		return fmt.Errorf("kubeconfig file not found in environment")
	}

	// Check if the kubeconfig file is the same as the one set by kubert,
	// if not, it means that the user or some other process has changed the KUBECONFIG environment variable
	// and kubert should not interfere with it, so kubert will choose to exit instead
	if kubertKubeconfig != kubeconfigPath {
		return fmt.Errorf("KUBECONFIG environment variable does not match kubert kubeconfig," +
			" to prevent kubert from interfering with your original kubeconfigs, please start a new shell with kubert")
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
