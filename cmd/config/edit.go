package configcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/idebeijer/kubert/internal/config"
)

func NewEditCommand() *cobra.Command {
	var create bool

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit the kubert config file in your editor",
		Long: `Open the kubert config file in your preferred editor for editing.

The editor is chosen from the $EDITOR or $VISUAL environment variable, falling back to vim.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var configFile string

			if create {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home directory: %w", err)
				}

				configDir := filepath.Join(home, ".config", "kubert")
				configFile = filepath.Join(configDir, "config.yaml")

				if _, err := os.Stat(configFile); err == nil {
					return fmt.Errorf("config file already exists at %s", configFile)
				}

				if err := os.MkdirAll(configDir, 0o700); err != nil {
					return fmt.Errorf("failed to create config directory: %w", err)
				}

				defaultConfig, err := config.GenerateDefaultYAML()
				if err != nil {
					return fmt.Errorf("failed to generate default config: %w", err)
				}

				if err := os.WriteFile(configFile, []byte(defaultConfig), 0o600); err != nil {
					return fmt.Errorf("failed to write config file: %w", err)
				}

				fmt.Printf("Created config file at %s\n", configFile)
			} else {
				configFile = viper.ConfigFileUsed()
				if configFile == "" {
					return fmt.Errorf("no config file is being used, create one with the '--create' flag")
				}
			}

			editor := os.Getenv("VISUAL")
			if editor == "" {
				editor = os.Getenv("EDITOR")
			}
			if editor == "" {
				editor = "vim"
			}

			safeEditor, err := exec.LookPath(editor)
			if err != nil {
				return fmt.Errorf("failed to find editor (%s): %w", editor, err)
			}

			// #nosec G702 -- editor is checked for existence with exec.LookPath
			editorCmd := exec.Command(safeEditor, configFile)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("failed to open editor (%s): %w", editor, err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&create, "create", false, "create a new config file at $HOME/.config/kubert/config.yaml with default settings")

	return cmd
}
