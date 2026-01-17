package protection

import (
	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/kubert"
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
			return runSetProtection(true)
		},
	}

	return cmd
}
