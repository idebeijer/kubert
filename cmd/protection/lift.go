package protection

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
)

func NewLiftCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lift <duration>",
		Short: "Temporarily lift protection for a duration",
		Long: `Temporarily lift protection for the current context.

The duration argument is required and specifies how long protection should be lifted.
Examples: 5m (5 minutes), 1h (1 hour), 30s (30 seconds)

After the duration expires, protection will automatically be restored.`,
		Example: `  # Lift protection for 5 minutes
  kubert protection lift 5m`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return kubert.ShellPreFlightCheck()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			duration, err := time.ParseDuration(args[0])
			if err != nil {
				return fmt.Errorf("invalid duration %q: %w", args[0], err)
			}

			sm, err := state.NewManager()
			if err != nil {
				return err
			}

			clientConfig, err := util.KubeClientConfig()
			if err != nil {
				return err
			}

			until := time.Now().Add(duration)
			if err := sm.LiftContextProtection(clientConfig.CurrentContext, until); err != nil {
				return err
			}

			fmt.Printf("Protection lifted for context %q until %s\n", clientConfig.CurrentContext, until.Format(time.RFC3339))
			return nil
		},
	}

	return cmd
}
