package kubeconfig

import (
	"fmt"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/kubeconfig"
	"github.com/spf13/cobra"
)

func addListFlags(cmd *cobra.Command) {}

func NewListCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "list",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Cfg

			fsProvider := kubeconfig.NewFileSystemProvider(cfg.KubeconfigPaths.Include, cfg.KubeconfigPaths.Exclude)

			loader := kubeconfig.NewLoader(fsProvider)

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
