package contextprotection

import (
	"fmt"
	"regexp"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
	"github.com/spf13/cobra"
)

func NewInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show protection status for the current context",
		Long: `Show protection status for the current context.

This will show if the current context is protected or not. If it is protected, it will show the explicit setting, otherwise it will show the default.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return kubert.ShellPreFlightCheck()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Cfg
			sm, err := state.NewManager()
			if err != nil {
				return err
			}

			clientConfig, err := util.KubeClientConfig()
			if err != nil {
				return err
			}

			if err := protectionStatus(sm, clientConfig.CurrentContext, cfg); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func protectionStatus(sm *state.Manager, context string, cfg config.Config) error {
	contextInfo, _ := sm.ContextInfo(context)
	if contextInfo.Protected == nil && cfg.Contexts.ProtectedByDefaultRegexp != nil {
		regex, err := regexp.Compile(*cfg.Contexts.ProtectedByDefaultRegexp)
		if err != nil {
			return fmt.Errorf("failed to compile regex: %w", err)
		}

		if regex.MatchString(context) {
			fmt.Println("Current context is protected: protection setting not set, but context matches protected by default regex")
		}
	}

	if contextInfo.Protected != nil && *contextInfo.Protected == true {
		fmt.Println("Current context is protected: explicit protect setting set")
	}

	if contextInfo.Protected != nil && *contextInfo.Protected == false {
		fmt.Println("Current context is unprotected: explicit unprotect setting set")
	}

	if contextInfo.Protected == nil && cfg.Contexts.ProtectedByDefaultRegexp == nil {
		fmt.Println("Current context is unprotected: no protection setting set")
	}

	return nil
}
