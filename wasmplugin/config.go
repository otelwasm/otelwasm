package wasmplugin

import "fmt"

// PluginConfig is a generic configuration type that can be passed to WASM modules
type PluginConfig map[string]interface{}

// Runtime type constants
const (
	RuntimeTypeWazero   = "wazero"
	RuntimeTypeWasmtime = "wasmtime"
)

// WazeroRuntimeMode represents the execution mode for Wazero runtime
type WazeroRuntimeMode string

const (
	// WazeroRuntimeModeInterpreter requests the WASM runtime to use the interpreter
	// mode for executing the WASM module.
	// This mode is basically slower than the compiled mode, but this is the
	// best option for most cases as it's portable and stable.
	// This is the default mode if not specified.
	WazeroRuntimeModeInterpreter WazeroRuntimeMode = "interpreter"

	// WazeroRuntimeModeCompiled requests the WASM runtime to use the compiled
	// mode for executing the WASM module.
	// This mode is faster than the interpreter mode, but it can only be used
	// on the supported platforms and architectures.
	// This mode is currently experimental as it doesn't work on all wasm
	// modules.
	// If the underlying platform and architecture is not supported, the
	// runtime will return an error.
	WazeroRuntimeModeCompiled WazeroRuntimeMode = "compiled"
)

// WasmtimeStrategy represents the compilation strategy for Wasmtime runtime
type WasmtimeStrategy string

const (
	WasmtimeStrategyCranelift WasmtimeStrategy = "cranelift"
)

// RuntimeConfig is the configuration for the WASM plugin runtime.
type RuntimeConfig struct {
	// Type specifies the Wasm runtime to use
	Type string `mapstructure:"type"`

	// Runtime-specific configurations
	Wazero   *WazeroConfig   `mapstructure:"wazero,omitempty"`
	Wasmtime *WasmtimeConfig `mapstructure:"wasmtime,omitempty"`

	// Remaining holds unknown runtime configurations for future extensibility
	Remaining map[string]interface{} `mapstructure:",remain"`
}

// WazeroConfig holds wazero-specific configurations
type WazeroConfig struct {
	// Mode is the runtime mode (interpreter or compiled) - existing feature
	Mode WazeroRuntimeMode `mapstructure:"mode,omitempty"`
}

// WasmtimeConfig holds wasmtime-specific configurations (minimal placeholder)
type WasmtimeConfig struct {
	// Strategy specifies compilation strategy
	Strategy WasmtimeStrategy `mapstructure:"strategy,omitempty"`
}

func (cfg *RuntimeConfig) Validate() error {
	supportedTypes := []string{RuntimeTypeWazero, RuntimeTypeWasmtime}

	if cfg.Type != "" {
		for _, supported := range supportedTypes {
			if cfg.Type == supported {
				return cfg.validateSpecific()
			}
		}
		return fmt.Errorf("unsupported runtime type: %s", cfg.Type)
	}
	return nil
}

func (cfg *RuntimeConfig) validateSpecific() error {
	switch cfg.Type {
	case RuntimeTypeWazero:
		if cfg.Wazero != nil {
			return cfg.Wazero.Validate()
		}
	case RuntimeTypeWasmtime:
		if cfg.Wasmtime != nil {
			return cfg.Wasmtime.Validate()
		}
	}
	return nil
}

func (cfg *WazeroConfig) Validate() error {
	if cfg.Mode != "" && cfg.Mode != WazeroRuntimeModeInterpreter && cfg.Mode != WazeroRuntimeModeCompiled {
		return fmt.Errorf("invalid wazero runtime mode: %s", cfg.Mode)
	}
	return nil
}

func (cfg *WasmtimeConfig) Validate() error {
	if cfg.Strategy != "" && cfg.Strategy != WasmtimeStrategyCranelift {
		return fmt.Errorf("invalid wasmtime strategy: %s", cfg.Strategy)
	}
	return nil
}

// Default sets the default values for the runtime configuration
// if they are not set.
func (cfg *RuntimeConfig) Default() {
	// Default to wazero if not specified
	if cfg.Type == "" {
		cfg.Type = RuntimeTypeWazero
	}

	// Set runtime-specific defaults
	switch cfg.Type {
	case RuntimeTypeWazero:
		if cfg.Wazero == nil {
			cfg.Wazero = &WazeroConfig{}
		}
		if cfg.Wazero.Mode == "" {
			cfg.Wazero.Mode = WazeroRuntimeModeInterpreter
		}
	case RuntimeTypeWasmtime:
		if cfg.Wasmtime == nil {
			cfg.Wasmtime = &WasmtimeConfig{}
		}
		if cfg.Wasmtime.Strategy == "" {
			cfg.Wasmtime.Strategy = WasmtimeStrategyCranelift
		}
	}
}

// DefaultRuntimeConfig is the default configuration for the WASM plugin runtime.
var DefaultRuntimeConfig = RuntimeConfig{
	Type: RuntimeTypeWazero,
	Wazero: &WazeroConfig{
		Mode: WazeroRuntimeModeInterpreter,
	},
}

// Config defines the common configuration for WASM components
type Config struct {
	// Path to the WASM module file
	Path string `mapstructure:"path"`

	// PluginConfig is the configuration to be passed to the WASM module
	PluginConfig PluginConfig `mapstructure:"plugin_config"`

	// RuntimeConfig is the configuration of WASM plugin runtime.
	RuntimeConfig *RuntimeConfig `mapstructure:"runtime_config,omitempty"`
}

// Validate validates the configuration
func (cfg *Config) Validate() error {
	if cfg.RuntimeConfig != nil {
		if err := cfg.RuntimeConfig.Validate(); err != nil {
			return err
		}
	}

	if cfg.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

// Default sets the default values for the configuration
// if they are not set.
func (cfg *Config) Default() {
	if cfg.RuntimeConfig == nil {
		cfg.RuntimeConfig = &DefaultRuntimeConfig
	} else {
		cfg.RuntimeConfig.Default()
	}
}
