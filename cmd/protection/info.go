package protection

import (
	"fmt"
	"regexp"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/idebeijer/kubert/internal/kubert"
	"github.com/idebeijer/kubert/internal/state"
	"github.com/idebeijer/kubert/internal/util"
)

func NewInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show protection status for current context",
		Long:  `Show the protection status for the current context, including explicit overrides and lift status.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return kubert.ShellPreFlightCheck()
		},
		RunE: runInfo,
	}

	cmd.Flags().StringP("output", "o", "", "Output format (short)")
	return cmd
}

func runInfo(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	if output != "" && output != "short" {
		return fmt.Errorf("invalid output format: %s", output)
	}

	cfg := config.Cfg
	sm, err := state.NewManager()
	if err != nil {
		return err
	}

	clientConfig, err := util.KubeClientConfig()
	if err != nil {
		return err
	}

	return protectionStatus(sm, clientConfig.CurrentContext, cfg, output)
}

func protectionStatus(sm *state.Manager, context string, cfg config.Config, output string) error {
	contextInfo, _ := sm.ContextInfo(context)

	yellow := color.New(color.FgHiYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	isShort := output == "short"

	if !isShort {
		fmt.Printf("Context: %s\n\n", cyan(context))
	}

	// Check for active lift
	if contextInfo.ProtectedUntil != nil {
		if time.Now().Before(*contextInfo.ProtectedUntil) {
			if isShort {
				fmt.Println("lifted")
			} else {
				fmt.Printf("%s Status: %s until %s\n", yellow("⏳"), yellow("LIFTED"), contextInfo.ProtectedUntil.Format(time.RFC3339))
				fmt.Printf("   Remaining: %s\n", time.Until(*contextInfo.ProtectedUntil).Round(time.Second))
			}
			return nil
		}
	}

	// Check explicit protection setting
	if contextInfo.Protected != nil {
		if *contextInfo.Protected {
			if isShort {
				fmt.Println("protected")
			} else {
				fmt.Printf("%s Status: %s (explicit override)\n", red("🔒"), red("PROTECTED"))
			}
		} else {
			if isShort {
				fmt.Println("unprotected")
			} else {
				fmt.Printf("%s Status: %s (explicit override)\n", green("🔓"), green("UNPROTECTED"))
			}
		}
		if !isShort {
			fmt.Println("   Use 'kubert protection remove' to revert to default")
		}
		return nil
	}

	// Check regex-based default
	if cfg.Protection.Regex != nil {
		regex, err := regexp.Compile(*cfg.Protection.Regex)
		if err != nil {
			return fmt.Errorf("failed to compile regex: %w", err)
		}

		if regex.MatchString(context) {
			if isShort {
				fmt.Println("protected")
			} else {
				fmt.Printf("%s Status: %s (matches default regex)\n", red("🔒"), red("PROTECTED"))
				fmt.Printf("   Regex: %s\n", *cfg.Protection.Regex)
			}
		} else {
			if isShort {
				fmt.Println("unprotected")
			} else {
				fmt.Printf("%s Status: %s (does not match default regex)\n", green("🔓"), green("UNPROTECTED"))
			}
		}
		return nil
	}

	if isShort {
		fmt.Println("unprotected")
	} else {
		fmt.Printf("%s Status: %s (no protection configured)\n", green("🔓"), green("UNPROTECTED"))
	}
	return nil
}
