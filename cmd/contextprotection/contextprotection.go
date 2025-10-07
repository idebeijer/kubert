package contextprotection

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context-protection",
		Short: "Protect and unprotect contexts",
		Long: `Protect and unprotect contexts.

This will allow you to protect and unprotect contexts for the "kubert kubectl" command. This can be useful if you want to prevent accidentally running certain kubectl commands to a cluster.
Keep in mind, this will only work when using kubectl through the "kubert kubectl" command. Direct commands using just "kubectl" will not be blocked. (If you use this feature, you could set an alias e.g. "k" for "kubert kubectl".)

Both "protect" and "unprotect" will set an explicit setting for the given context. That means if either of those has been set, kubert will ignore the default setting. If you want to use the default setting again, use "kubert context-protection delete <context>".

What kubectl commands should be blocked can be configured in the kubert configuration file.`,
		Aliases: []string{"ctx-protection"},
	}

	cmd.AddCommand(NewProtectCommand())
	cmd.AddCommand(NewUnprotectCommand())
	cmd.AddCommand(NewDeleteCommand())
	cmd.AddCommand(NewInfoCommand())

	return cmd
}
