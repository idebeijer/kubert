package contextprotection

import (
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
	"github.com/spf13/cobra"
)

func NewProtectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protect",
		Short: "Protect current context",
		Long: `Protect current context.

This will set an explicit "protect" for the current context. That means it wil override the default setting. If the current context should use the default again, use "kubert context-protection delete".`,
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

			return nil
		},
	}

	return cmd
}
