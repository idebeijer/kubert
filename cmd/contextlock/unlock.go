package contextlock

import (
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
	"github.com/spf13/cobra"
)

func NewUnlockCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock current context",
		Long: `Unlock current context. 

This will set an explicit "unlock" for the current context. That means it wil override the default setting. If the current context should use the default again, use "kubert context-lock delete".`,
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

			if err := sm.SetContextLock(clientConfig.CurrentContext, false); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
