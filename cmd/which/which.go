package which

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "which",
		Short: "Display information about current context, cluster, namespace, or config",
		Long:  `Display information about the current Kubernetes context, cluster, namespace, or kubert config file.`,
	}

	cmd.AddCommand(newContextCommand())
	cmd.AddCommand(newNamespaceCommand())
	cmd.AddCommand(newConfigCommand())
	cmd.AddCommand(newClusterCommand())

	return cmd
}
