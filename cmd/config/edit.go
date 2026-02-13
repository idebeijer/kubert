package configcmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit the kubert config file in vim",
		Long:  `Open the kubert config file in vim editor for editing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := viper.ConfigFileUsed()
			if configFile == "" {
				return fmt.Errorf("no config file is being used")
			}

			vimCmd := exec.Command("vim", configFile)
			vimCmd.Stdin = os.Stdin
			vimCmd.Stdout = os.Stdout
			vimCmd.Stderr = os.Stderr

			if err := vimCmd.Run(); err != nil {
				return fmt.Errorf("failed to open vim: %w", err)
			}

			return nil
		},
	}
}
