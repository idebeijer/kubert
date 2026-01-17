package protection

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
)

func NewRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove explicit protection override",
		Long: `Remove any explicit protection override for the current context.

This clears both the explicit protected/unprotected setting and any active lift,
reverting the context to use the default regex-based protection from config.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return kubert.ShellPreFlightCheck()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			sm, err := state.NewManager()
			if err != nil {
				return err
			}

			clientConfig, err := util.KubeClientConfig()
			if err != nil {
				return err
			}

			if err := sm.DeleteContextProtection(clientConfig.CurrentContext); err != nil {
				return err
			}

			// Also clear any active lift
			if err := sm.ClearProtectedUntil(clientConfig.CurrentContext); err != nil {
				return fmt.Errorf("failed to clear active lift: %w", err)
			}

			fmt.Printf("Removed protection override for context %q (now using default regex)\n", clientConfig.CurrentContext)
			return nil
		},
	}

	return cmd
}
