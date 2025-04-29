package wasmplugin

import "fmt"

// PluginConfig is a generic configuration type that can be passed to WASM modules
type PluginConfig map[string]interface{}

type RuntimeMode string

const (
	// RuntimeModeInterpreter requests the WASM runtime to use the interpreter
	// mode for executing the WASM module.
	// This mode is basically slower than the compiled mode, but this is the
	// best option for most cases as it's portable and stable.
	// This is the default mode if not specified.
	RuntimeModeInterpreter RuntimeMode = "interpreter"

	// RuntimeModeCompiled requests the WASM runtime to use the compiled
	// mode for executing the WASM module.
	// This mode is faster than the interpreter mode, but it can only be used
	// on the supported platforms and architectures.
	// This mode is currently experimental as it doesn't work on all wasm
	// modules.
	// If the underlying platform and architecture is not supported, the
	// runtime will return an error.
	RuntimeModeCompiled RuntimeMode = "compiled"
)

// RuntimeConfig is the configuration for the WASM plugin runtime.
type RuntimeConfig struct {
	// Mode is the runtime mode for the WASM plugin.
	// The default is "interpreter".
	Mode RuntimeMode `mapstructure:"mode,omitempty"`
}

func (cfg *RuntimeConfig) Validate() error {
	if cfg.Mode != RuntimeModeInterpreter && cfg.Mode != RuntimeModeCompiled {
		return fmt.Errorf("invalid runtime mode: %s", cfg.Mode)
	}
	return nil
}

// Default sets the default values for the runtime configuration
// if they are not set.
func (cfg *RuntimeConfig) Default() {
	if cfg.Mode == "" {
		cfg.Mode = DefaultRuntimeConfig.Mode
	}
}

// DefaultRuntimeConfig is the default configuration for the WASM plugin runtime.
var DefaultRuntimeConfig = RuntimeConfig{
	Mode: RuntimeModeInterpreter,
}

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
	if err := cfg.RuntimeConfig.Validate(); err != nil {
		return err
	}

	if cfg.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

// Default sets the default values for the configuration
// if they are not set.
func (cfg *Config) Default() {
	cfg.RuntimeConfig.Default()
}
