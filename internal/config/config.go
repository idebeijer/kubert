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
	// DefaultLocked is a regex that matches contexts that should be locked by default.
	DefaultLocked *string `mapstructure:"defaultLocked" yaml:"defaultLocked"`

	// BlockedKubectlCommands is a list of kubectl commands that should be blocked when the context is locked.
	BlockedKubectlCommands []string `mapstructure:"blockedKubectlCommands" yaml:"blockedKubectlCommands"`
}

func init() {
	viper.SetDefault("kubeconfigs.include", []string{
		"~/.kube/config",
	})
	viper.SetDefault("kubeconfigs.exclude", []string{})
	viper.SetDefault("interactiveShellMode", true)
	viper.SetDefault("defaultLockedContexts", []string{})
	viper.SetDefault("contexts.defaultLocked", nil)
	viper.SetDefault("contexts.blockedKubectlCommands", []string{
		"delete",
		"edit",
		"exec",
		"drain",
		"cordon",
		"uncordon",
		"scale",
		"replace",
		"apply",
		"patch",
		"set",
	})
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
