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
	isShort := output == "short"

	if !isShort {
		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Printf("Context: %s\n\n", cyan(context))
	}

	if hasActiveLift(contextInfo) {
		printLiftedStatus(contextInfo, isShort)
		return nil
	}

	if contextInfo.Protected != nil {
		printExplicitOverride(*contextInfo.Protected, isShort)
		return nil
	}

	if cfg.Protection.Regex != nil {
		return printRegexStatus(context, *cfg.Protection.Regex, isShort)
	}

	printStatus(true, "no protection configured", isShort)
	return nil
}

func hasActiveLift(info state.ContextInfo) bool {
	return info.ProtectedUntil != nil && time.Now().Before(*info.ProtectedUntil)
}

func printLiftedStatus(info state.ContextInfo, short bool) {
	if short {
		fmt.Println("lifted")
		return
	}

	yellow := color.New(color.FgHiYellow).SprintFunc()
	fmt.Printf("%s Status: %s until %s\n", yellow("⏳"), yellow("LIFTED"), info.ProtectedUntil.Format(time.RFC3339))
	fmt.Printf("   Remaining: %s\n", time.Until(*info.ProtectedUntil).Round(time.Second))
}

func printExplicitOverride(protected bool, short bool) {
	printStatus(!protected, "explicit override", short)
	if !short {
		fmt.Println("   Use 'kubert protection remove' to revert to default")
	}
}

func printRegexStatus(context, pattern string, short bool) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("failed to compile regex: %w", err)
	}

	if regex.MatchString(context) {
		printStatus(false, "matches default regex", short)
		if !short {
			fmt.Printf("   Regex: %s\n", pattern)
		}
	} else {
		printStatus(true, "does not match default regex", short)
	}
	return nil
}

func printStatus(unprotected bool, reason string, short bool) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	if unprotected {
		if short {
			fmt.Println("unprotected")
			return
		}
		fmt.Printf("%s Status: %s (%s)\n", green("🔓"), green("UNPROTECTED"), reason)
		return
	}

	if short {
		fmt.Println("protected")
		return
	}
	fmt.Printf("%s Status: %s (%s)\n", red("🔒"), red("PROTECTED"), reason)
}
