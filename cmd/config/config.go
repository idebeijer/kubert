package configcmd

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage kubert configuration",
		Long:  `Manage kubert configuration file.`,
	}

	cmd.AddCommand(NewEditCommand())

	return cmd
}
