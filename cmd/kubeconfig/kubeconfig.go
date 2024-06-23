package kubeconfig

import (
	"github.com/spf13/cobra"
)

func initFlags(cmd *cobra.Command) {
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{}

	cmd = &cobra.Command{
		Use:   "kubeconfig",
		Short: "kubeconfig command",
	}

	initFlags(cmd)

	cmd.AddCommand(NewListCommand())

	return cmd
}
