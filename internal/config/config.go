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
	Protection           Protection      `mapstructure:"protection" yaml:"protection"`
	Hooks                Hooks           `mapstructure:"hooks" yaml:"hooks"`
	Fzf                  Fzf             `mapstructure:"fzf" yaml:"fzf"`
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

type Protection struct {
	// Regex is a regular expression that matches contexts that should be protected by default.
	Regex *string `mapstructure:"regex" yaml:"regex"`

	// Commands is a list of kubectl commands that should be blocked when the context is protected.
	Commands []string `mapstructure:"commands" yaml:"commands"`

	// Prompt enables the confirmation prompt before running protected commands.
	// If false, kubert will immediately exit when a protected command is run.
	Prompt bool `mapstructure:"prompt" yaml:"prompt"`
}

type Hooks struct {
	// PreShell is a shell command that will be executed before spawning the shell with the selected context.
	PreShell string `mapstructure:"preShell" yaml:"preShell"`

	// PostShell is a shell command that will be executed after exiting the shell with the selected context.
	PostShell string `mapstructure:"postShell" yaml:"postShell"`
}

type Fzf struct {
	// Opts are additional options passed to fzf when selecting contexts or namespaces.
	Opts string `mapstructure:"opts" yaml:"opts"`
}

func init() {
	viper.SetDefault("kubeconfigs.include", []string{
		"~/.kube/config",
		"~/.kube/*.yml",
		"~/.kube/*.yaml",
	})
	viper.SetDefault("kubeconfigs.exclude", []string{})
	viper.SetDefault("interactiveShellMode", true)
	viper.SetDefault("protection.regex", nil)
	viper.SetDefault("protection.commands", []string{
		"delete",
		"edit",
		"exec",
		"drain",
		"scale",
		"autoscale",
		"replace",
		"apply",
		"patch",
		"set",
	})
	viper.SetDefault("protection.prompt", true)
	viper.SetDefault("hooks.preShell", "")
	viper.SetDefault("hooks.postShell", "")
	viper.SetDefault("fzf.opts", "")
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
