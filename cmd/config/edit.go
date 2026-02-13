package configcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewEditCommand() *cobra.Command {
	var create bool

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit the kubert config file in vim",
		Long:  `Open the kubert config file in vim editor for editing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var configFile string

			if create {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home directory: %w", err)
				}

				configDir := filepath.Join(home, ".config", "kubert")
				configFile = filepath.Join(configDir, "config.yaml")

				// Check if file already exists
				if _, err := os.Stat(configFile); err == nil {
					return fmt.Errorf("config file already exists at %s", configFile)
				}

				// Create the kubert directory if it doesn't exist
				if err := os.MkdirAll(configDir, 0o700); err != nil {
					return fmt.Errorf("failed to create config directory: %w", err)
				}

				// Create empty config file TODO: maybe add default config?
				if err := os.WriteFile(configFile, []byte{}, 0o600); err != nil {
					return fmt.Errorf("failed to write config file: %w", err)
				}

				fmt.Printf("Created config file at %s\n", configFile)
			} else {
				configFile = viper.ConfigFileUsed()
				if configFile == "" {
					return fmt.Errorf("no config file is being used, create one with the '--create' flag")
				}
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

	cmd.Flags().BoolVar(&create, "create", false, "Create a new config file at $HOME/.config/kubert/config.yaml")

	return cmd
}
