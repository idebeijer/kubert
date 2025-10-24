package which

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/util"
)

func newContextCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "ctx",
		Aliases: []string{"context"},
		Short:   "Display the current Kubernetes context",
		Long:    `Display the current Kubernetes context.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientConfig, err := util.KubeClientConfig()
			if err != nil {
				return fmt.Errorf("failed to load kubeconfig: %w", err)
			}

			if clientConfig.CurrentContext == "" {
				return fmt.Errorf("no current context set")
			}

			fmt.Println(clientConfig.CurrentContext)
			return nil
		},
	}
}
