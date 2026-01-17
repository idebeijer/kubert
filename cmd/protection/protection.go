package protection

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
)

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

func runSetProtection(protect bool) error {
	sm, err := state.NewManager()
	if err != nil {
		return err
	}

	clientConfig, err := util.KubeClientConfig()
	if err != nil {
		return err
	}

	if err := sm.SetContextProtection(clientConfig.CurrentContext, protect); err != nil {
		return err
	}

	// Clear any active lift (best effort, ignore errors since main operation succeeded)
	_ = sm.ClearProtectedUntil(clientConfig.CurrentContext)

	status := "unprotected"
	if protect {
		status = "protected"
	}
	fmt.Printf("Context %q is now %s\n", clientConfig.CurrentContext, status)
	return nil
}
