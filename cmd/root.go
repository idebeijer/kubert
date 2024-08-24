package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/idebeijer/kubert/cmd/contextprotection"
	"github.com/idebeijer/kubert/cmd/kubeconfig"
	"github.com/idebeijer/kubert/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
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

	c.PersistentFlags().StringVar(&c.cfgFile, "config", "", "config file (default is $HOME/.kubert/config.yaml)")
}

func (c *RootCmd) addCommands() {
	c.AddCommand(kubeconfig.NewCommand())
	c.AddCommand(contextprotection.NewCommand())
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

		// Search config in home directory
		viper.AddConfigPath(filepath.Join(home, ".kubert"))
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
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
