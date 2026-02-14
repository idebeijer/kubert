package kubeconfig

import (
	"github.com/spf13/cobra"
)

func initFlags(cmd *cobra.Command) {
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeconfig",
		Short: "Manage and inspect kubeconfig files",
	}

	initFlags(cmd)

	cmd.AddCommand(NewListCommand())
	cmd.AddCommand(newLintCommand())

	return cmd
}
