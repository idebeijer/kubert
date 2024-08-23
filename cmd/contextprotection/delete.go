package contextprotection

import (
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete protection setting for the current context",
		Long: `Delete protection setting for the current context.

This will delete the explicit protect/unprotect setting for the current context. So if either "protect" or "unprotect" was set, it will be removed and the default will be used.`,
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

			return nil
		},
	}

	return cmd
}
