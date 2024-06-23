package config

import "github.com/spf13/viper"

var Cfg Config

type Config struct {
	KubeconfigPaths      KubeconfigPaths `mapstructure:"kubeconfigs"`
	InteractiveShellMode bool            `mapstructure:"interactiveShellMode"`
}

type KubeconfigPaths struct {
	Include []string `mapstructure:"include"`
	Exclude []string `mapstructure:"exclude"`
}

type KubeconfigProvider struct {
	Local       LocalKubeconfigProvider `mapstructure:"local"`
	OnePassword OnePasswordProvider     `mapstructure:"onepassword"`
}

type LocalKubeconfigProvider struct {
	Enabled bool `mapstructure:"enabled"`
}

type OnePasswordProvider struct {
	Enabled bool `mapstructure:"enabled"`
}

func init() {
	viper.SetDefault("kubeconfigs.include", []string{
		"~/.kube/config",
	})
	viper.SetDefault("kubeconfigs.exclude", []string{})
	viper.SetDefault("interactiveShellMode", true)
}
