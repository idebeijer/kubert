package contextlock

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{}

	cmd = &cobra.Command{
		Use:   "context-lock",
		Short: "Lock and unlock contexts",
		Long: `Lock and unlock contexts.

This will allow you to lock and unlock contexts for the "kubert kubectl" command. This can be useful if you want to prevent accidentally running certain kubectl commands to a cluster.
Keep in mind, this will only work when using kubectl through the "kubert kubectl" command. Direct commands using just "kubectl" will not be blocked. (If you use this feature, you could set an alias for "kubectl" or "k" to "kubert kubectl".)

Both "lock" and "unlock" will set an explicit setting for the given context. That means if either of those has been set, kubert will ignore the default setting. If you want to use the default setting again, use "kubert context-lock delete <context>".

What kubectl commands should be blocked can be configured in the kubert configuration file.`,
	}

	cmd.AddCommand(NewLockCommand())
	cmd.AddCommand(NewUnlockCommand())
	cmd.AddCommand(NewDeleteCommand())

	return cmd
}
