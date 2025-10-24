package config

import (
	"fmt"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var Cfg Config

type Config struct {
	KubeconfigPaths      KubeconfigPaths `mapstructure:"kubeconfigs" yaml:"kubeconfigs"`
	InteractiveShellMode bool            `mapstructure:"interactiveShellMode" yaml:"interactiveShellMode"`
	Contexts             Contexts        `mapstructure:"contexts" yaml:"contexts"`
	Hooks                Hooks           `mapstructure:"hooks" yaml:"hooks"`
}

type KubeconfigPaths struct {
	Include []string `mapstructure:"include" yaml:"include"`
	Exclude []string `mapstructure:"exclude" yaml:"exclude"`
}

type KubeconfigProvider struct {
	Local LocalKubeconfigProvider `mapstructure:"local" yaml:"local"`
}

type LocalKubeconfigProvider struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
}

type Contexts struct {
	// ProtectedByDefaultRegexp is a regex that matches contexts that should be protected by default.
	ProtectedByDefaultRegexp *string `mapstructure:"protectedByDefaultRegexp" yaml:"protectedByDefaultRegexp"`

	// ProtectedKubectlCommands is a list of kubectl commands that should be blocked when the context is protected.
	ProtectedKubectlCommands []string `mapstructure:"protectedKubectlCommands" yaml:"protectedKubectlCommands"`

	// ExitOnProtectedKubectlCmd disables the default confirmation prompt and instead immediately exits out if the context is protected.
	ExitOnProtectedKubectlCmd bool `mapstructure:"exitOnProtectedKubectlCmd" yaml:"exitOnProtectedKubectlCmd"`
}

type Hooks struct {
	// PreShell is a shell command that will be executed before spawning the shell with the selected context.
	PreShell string `mapstructure:"preShell" yaml:"preShell"`

	// PostShell is a shell command that will be executed after exiting the shell with the selected context.
	PostShell string `mapstructure:"postShell" yaml:"postShell"`
}

func init() {
	viper.SetDefault("kubeconfigs.include", []string{
		"~/.kube/config",
	})
	viper.SetDefault("kubeconfigs.exclude", []string{})
	viper.SetDefault("interactiveShellMode", true)
	viper.SetDefault("contexts.protectedByDefaultRegexp", nil)
	viper.SetDefault("contexts.protectedKubectlCommands", []string{
		"delete",
		"edit",
		"exec",
		"drain",
		"cordon",
		"uncordon",
		"scale",
		"autoscale",
		"replace",
		"apply",
		"patch",
		"set",
	})
	viper.SetDefault("contexts.exitOnProtectedKubectlCmd", false)
	viper.SetDefault("hooks.preShell", "")
	viper.SetDefault("hooks.postShell", "")
}

func GenerateDefaultYAML() (string, error) {
	if err := viper.Unmarshal(&Cfg); err != nil {
		return "", fmt.Errorf("unable to unmarshal config: %w", err)
	}

	// Marshal the Config struct into YAML
	yamlData, err := yaml.Marshal(&Cfg)
	if err != nil {
		return "", fmt.Errorf("unable to marshal config to YAML: %w", err)
	}

	return string(yamlData), nil
}
