package which

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/util"
)

func newClusterCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cluster",
		Short: "Display the current Kubernetes cluster",
		Long:  `Display the current Kubernetes cluster name for the active context.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientConfig, err := util.KubeClientConfig()
			if err != nil {
				return fmt.Errorf("failed to load kubeconfig: %w", err)
			}

			if clientConfig.CurrentContext == "" {
				return fmt.Errorf("no current context set")
			}

			ctx, ok := clientConfig.Contexts[clientConfig.CurrentContext]
			if !ok {
				return fmt.Errorf("current context %q not found in kubeconfig", clientConfig.CurrentContext)
			}

			fmt.Println(ctx.Cluster)
			return nil
		},
	}
}
