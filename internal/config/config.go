package config

import (
	"bytes"
	"fmt"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v4"
)

var (
	Cfg        Config // Global 'current' config
	DefaultCfg Config // Pure default config, captured before reading any config file
)

type Config struct {
	KubeconfigPaths KubeconfigPaths `mapstructure:"kubeconfigs" yaml:"kubeconfigs"`
	// Deprecated: use Interactive instead.
	InteractiveShellMode bool       `mapstructure:"interactiveShellMode" yaml:"interactiveShellMode,omitempty"`
	Interactive          bool       `mapstructure:"interactive" yaml:"interactive"`
	Recursive            bool       `mapstructure:"recursive" yaml:"recursive"`
	Protection           Protection `mapstructure:"protection" yaml:"protection"`
	Hooks                Hooks      `mapstructure:"hooks" yaml:"hooks"`
	Fzf                  Fzf        `mapstructure:"fzf" yaml:"fzf"`
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

func setDefaults() {
	viper.SetDefault("kubeconfigs.include", []string{
		"~/.kube/config",
		"~/.kube/*.yml",
		"~/.kube/*.yaml",
	})
	viper.SetDefault("kubeconfigs.exclude", []string{})
	viper.SetDefault("interactive", true)
	viper.SetDefault("recursive", false)
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

func init() {
	setDefaults()

	// Capture the pure defaults immediately into DefaultCfg
	if err := viper.Unmarshal(&DefaultCfg); err != nil {
		panic(fmt.Errorf("failed to capture default config: %w", err))
	}
}

// GenerateDefaultYAML returns the default configuration as a YAML string, without any user overrides.
func GenerateDefaultYAML() (string, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(&DefaultCfg); err != nil {
		return "", fmt.Errorf("unable to marshal config to YAML: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return "", fmt.Errorf("unable to close YAML encoder: %w", err)
	}

	return buf.String(), nil
}
