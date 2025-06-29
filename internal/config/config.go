package config

import (
	"fmt"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var Cfg Config

type Config struct {
	KubeconfigPaths      KubeconfigPaths     `mapstructure:"kubeconfigs" yaml:"kubeconfigs"`
	KubeconfigProviders  KubeconfigProviders `mapstructure:"providers" yaml:"providers"`
	InteractiveShellMode bool                `mapstructure:"interactiveShellMode" yaml:"interactiveShellMode"`
	Contexts             Contexts            `mapstructure:"contexts" yaml:"contexts"`
}

type KubeconfigPaths struct {
	Include []string `mapstructure:"include" yaml:"include"`
	Exclude []string `mapstructure:"exclude" yaml:"exclude"`
}

type KubeconfigProviders struct {
	Filesystem     FilesystemProvider     `mapstructure:"filesystem" yaml:"filesystem"`
	MacOSEncrypted MacOSEncryptedProvider `mapstructure:"macosEncrypted" yaml:"macosEncrypted"`
}

type FilesystemProvider struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
}

type MacOSEncryptedProvider struct {
	Enabled    bool   `mapstructure:"enabled" yaml:"enabled"`
	StorageDir string `mapstructure:"storageDir" yaml:"storageDir"`
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

func init() {
	viper.SetDefault("kubeconfigs.include", []string{
		"~/.kube/config",
	})
	viper.SetDefault("kubeconfigs.exclude", []string{})
	viper.SetDefault("providers.filesystem.enabled", true)
	viper.SetDefault("providers.macosEncrypted.enabled", false)
	viper.SetDefault("providers.macosEncrypted.storageDir", "~/.kubert/encrypted")
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
