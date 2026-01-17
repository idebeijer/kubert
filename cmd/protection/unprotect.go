package protection

import (
	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/kubert"
)

func NewUnprotectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unprotect",
		Short: "Explicitly unprotect current context",
		Long: `Explicitly unprotect the current context.

This sets an explicit unprotected override for the current context.
To revert to the default regex-based protection, use "kubert protection remove".`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return kubert.ShellPreFlightCheck()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetProtection(false)
		},
	}

	return cmd
}
