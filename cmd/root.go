package cmd

import (
	"log/slog"
	"os"

	"github.com/idebeijer/kubert/cmd/kubeconfig"
	"github.com/idebeijer/kubert/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	// KubertShellActiveEnvVar is the environment variable that is set to indicate that Kubert is active.
	KubertShellActiveEnvVar = "KUBERT_SHELL_ACTIVE"

	// KubertShellKubeconfigEnvVar is the environment variable that is set to the path of the temporary kubeconfig file.
	KubertShellKubeconfigEnvVar = "KUBERT_SHELL_KUBECONFIG"
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

	c.PersistentFlags().Bool("debug", false, "debug mode")
	_ = viper.BindPFlag("debug", c.PersistentFlags().Lookup("debug"))

	c.PersistentFlags().StringVar(&c.cfgFile, "config", "", "config file (default is $HOME/.kubert.yaml)")
}

func (c *RootCmd) addCommands() {
	c.AddCommand(kubeconfig.NewCommand())
	c.AddCommand(NewContextCommand())
	c.AddCommand(NewNamespaceCommand())
	c.AddCommand(NewKubectlCommand())
}

func (c *RootCmd) initConfig() {
	var level slog.Level

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

	viper.SetEnvPrefix("kubert")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		//fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
	_ = viper.ReadInConfig()
	_ = viper.Unmarshal(&config.Cfg)

	if viper.GetBool("debug") {
		level = slog.LevelDebug
	}
	slog.SetLogLoggerLevel(level)
}
