package which

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Display the path to the kubert config file",
		Long:    `Display the path to the kubert config file if one is being used.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := viper.ConfigFileUsed()
			if configFile == "" {
				fmt.Println("No config file is being used")
				return nil
			}

			fmt.Println(configFile)
			return nil
		},
	}
}
