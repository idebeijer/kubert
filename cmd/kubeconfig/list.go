package kubeconfig

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/kubeconfig"
)

func addListFlags(cmd *cobra.Command) {}

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all kubeconfig files being used",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Cfg

			fsProvider := kubeconfig.NewFileSystemProvider(cfg.KubeconfigPaths.Include, cfg.KubeconfigPaths.Exclude)

			loader := kubeconfig.NewLoader(kubeconfig.WithProvider(fsProvider))

			kubeconfigs, err := loader.LoadAll()
			if err != nil {
				return err
			}

			for _, k8sconfig := range kubeconfigs {
				fmt.Println(k8sconfig.FilePath)
			}

			return nil
		},
	}

	addListFlags(cmd)

	return cmd
}
