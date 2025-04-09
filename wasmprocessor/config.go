package wasmprocessor

import "fmt"

type Config struct {
	Path string `mapstructure:"path"`

	PluginConfig PluginConfig `mapstructure:"plugin_config"`
}

func (cfg *Config) Validate() error {
	if cfg.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

type PluginConfig map[string]any
