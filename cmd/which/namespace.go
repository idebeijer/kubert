package which

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/util"
)

func newNamespaceCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "ns",
		Aliases: []string{"namespace"},
		Short:   "Display the current Kubernetes namespace",
		Long:    `Display the current Kubernetes namespace for the current context.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientConfig, err := util.KubeClientConfig()
			if err != nil {
				return fmt.Errorf("failed to load kubeconfig: %w", err)
			}

			if clientConfig.CurrentContext == "" {
				return fmt.Errorf("no current context set")
			}

			context, ok := clientConfig.Contexts[clientConfig.CurrentContext]
			if !ok {
				return fmt.Errorf("context %s not found", clientConfig.CurrentContext)
			}

			namespace := context.Namespace
			if namespace == "" {
				namespace = "default"
			}

			fmt.Println(namespace)
			return nil
		},
	}
}
