package protection

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protection",
		Short: "Manage context protection",
		Long: `Manage context protection for the current kubert shell.

Protection prevents accidentally running destructive kubectl commands in sensitive contexts.
This only works when using kubectl through "kubert kubectl" (consider aliasing k=kubert kubectl).`,
		Aliases: []string{"context-protection", "ctx-protection"},
	}

	cmd.AddCommand(NewProtectCommand())
	cmd.AddCommand(NewUnprotectCommand())
	cmd.AddCommand(NewLiftCommand())
	cmd.AddCommand(NewRemoveCommand())
	cmd.AddCommand(NewInfoCommand())

	return cmd
}
