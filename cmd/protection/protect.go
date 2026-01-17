package protection

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
)

func NewProtectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protect",
		Short: "Explicitly protect current context",
		Long: `Explicitly protect the current context.

This sets an explicit protection override for the current context.
To revert to the default regex-based protection, use "kubert protection remove".`,
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

			if err := sm.SetContextProtection(clientConfig.CurrentContext, true); err != nil {
				return err
			}

			// Clear any active lift
			_ = sm.ClearProtectedUntil(clientConfig.CurrentContext)

			fmt.Printf("Context %q is now protected\n", clientConfig.CurrentContext)
			return nil
		},
	}

	return cmd
}
