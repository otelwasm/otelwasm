package wasmplugin

import "fmt"

// PluginConfig is a generic configuration type that can be passed to WASM modules
type PluginConfig map[string]interface{}

// RuntimeConfig is the configuration for the WASM plugin runtime.
type RuntimeConfig struct{}

// Config defines the common configuration for WASM components
type Config struct {
	// Path to the WASM module file
	Path string `mapstructure:"path"`

	// PluginConfig is the configuration to be passed to the WASM module
	PluginConfig PluginConfig `mapstructure:"plugin_config"`

	// Runtime is the configuration of WASM plugin runtime.
	RuntimeConfig RuntimeConfig `mapstructure:"runtime"`
}

// Validate validates the configuration
func (cfg *Config) Validate() error {
	if cfg.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}
