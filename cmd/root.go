package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	configcmd "github.com/idebeijer/kubert/cmd/config"
	"github.com/idebeijer/kubert/cmd/kubeconfig"
	"github.com/idebeijer/kubert/cmd/protection"
	"github.com/idebeijer/kubert/cmd/which"
	"github.com/idebeijer/kubert/internal/config"
)

type RootCmd struct {
	*cobra.Command

	cfgFile string
}

func NewRootCmd() *RootCmd {
	cmd := &RootCmd{}
	cmd.Command = &cobra.Command{
		Use:   "kubert",
		Short: "kubert is a tool to switch kubernetes contexts and namespaces",
		Long: `kubert is a CLI tool to switch kubernetes contexts and namespaces within an isolated shell so you can have multiple shells with different contexts and namespaces.

It also includes a wrapper around kubectl to provide the ability to protect contexts by setting a regex pattern to match the context name. This can be used to prevent accidentally running certain kubectl commands in an unwanted context.
Keep in mind, this will only work when using kubectl through the "kubert kubectl" command. Direct commands using just "kubectl" will not be blocked. (If you use this feature, you could set an alias e.g. "k" for "kubert kubectl".)
`,
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

	c.PersistentFlags().StringVar(&c.cfgFile, "config", "", "config file (default is $HOME/.config/kubert/config.yaml, can be overridden by KUBERT_CONFIG)")
}

func (c *RootCmd) addCommands() {
	c.AddCommand(configcmd.NewCommand())
	c.AddCommand(kubeconfig.NewCommand())
	c.AddCommand(protection.NewCommand())
	c.AddCommand(NewContextCommand())
	c.AddCommand(NewNamespaceCommand())
	c.AddCommand(NewKubectlCommand())
	c.AddCommand(NewExecCommand())
	c.AddCommand(which.NewCommand())
	c.AddCommand(NewVersionCommand())
}

func (c *RootCmd) initConfig() {
	var level slog.Level

	if os.Getenv("KUBERT_CONFIG") != "" {
		viper.SetConfigFile(os.Getenv("KUBERT_CONFIG"))
	} else if c.cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(c.cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory
		viper.AddConfigPath(filepath.Join(home, ".config/kubert"))
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("kubert")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
		} else {
			slog.Error("Failed to read config file", "error", err)
			os.Exit(1)
		}
	} else {
		// Config loaded successfully
		slog.Debug("Using config file", "file", viper.ConfigFileUsed())
	}

	// Check if deprecated key was provided
	if viper.IsSet("interactiveShellMode") {
		fmt.Println("WARNING: 'interactiveShellMode' is deprecated. Please update your config to use 'interactive' instead.")

		// Map the old value to the new key
		viper.Set("interactive", viper.GetBool("interactiveShellMode"))
	}

	// Unmarshal config into struct
	if err := viper.Unmarshal(&config.Cfg); err != nil {
		slog.Error("Unable to decode config file", "error", err)
		os.Exit(1)
	}

	if viper.GetBool("debug") {
		level = slog.LevelDebug
	}
	slog.SetLogLoggerLevel(level)
}
