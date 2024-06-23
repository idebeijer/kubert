package cmd

import (
	"log/slog"
	"os"

	"github.com/idebeijer/kubert/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RootCmd struct {
	*cobra.Command

	cfgFile string
}

func NewRootCmd() *RootCmd {
	cmd := &RootCmd{}
	cmd.Command = &cobra.Command{
		Use:   "kubert",
		Short: "kubert",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.initFlags()
	cmd.addCommands()

	return cmd
}

func (c *RootCmd) Execute() {
	if err := c.Command.Execute(); err != nil {
		os.Exit(1)
	}
}

func (c *RootCmd) initFlags() {
	cobra.OnInitialize(c.initConfig)

	c.PersistentFlags().String("log-level", "info", "log level")
	viper.BindPFlag("log-level", c.PersistentFlags().Lookup("log-level"))

	c.PersistentFlags().StringVar(&c.cfgFile, "config", "", "config file (default is $HOME/.kubert.yaml)")
}

func (c *RootCmd) addCommands() {
}

func (c *RootCmd) initConfig() {
	var level slog.Level
	logLevel := viper.GetString("log-level")
	_ = level.UnmarshalText([]byte(logLevel))
	logHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	if c.cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(c.cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".kubert.yaml" (with extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kubert.yaml")
	}

	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		//fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
	_ = viper.ReadInConfig()
	_ = viper.Unmarshal(&config.Cfg)
}
