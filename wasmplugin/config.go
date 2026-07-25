package wasmplugin

import "fmt"

// PluginConfig is a generic configuration type that can be passed to WASM modules
type PluginConfig map[string]interface{}

// RuntimeName selects the WebAssembly runtime implementation used to
// execute the WASM plugin.
type RuntimeName string

const (
	// RuntimeNameWazero runs the WASM plugin on the wazero runtime.
	// wazero is a pure-Go runtime and requires no external dependencies.
	// This is the default runtime if not specified.
	RuntimeNameWazero RuntimeName = "wazero"

	// RuntimeNameWasmEdge runs the WASM plugin on the WasmEdge runtime.
	// WasmEdge support requires building with the "wasmedge" build tag and
	// the WasmEdge shared library installed on the system.
	RuntimeNameWasmEdge RuntimeName = "wasmedge"
)

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
	// For wazero, this uses the compiler backend; for wasmedge, this
	// AOT-compiles the module at startup.
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
	// Name is the WebAssembly runtime implementation to use.
	// Valid values are "wazero" (default) and "wasmedge".
	Name RuntimeName `mapstructure:"name,omitempty"`

	// Mode is the runtime mode for the WASM plugin.
	// The default is "interpreter".
	// This option only applies to the wazero runtime.
	Mode RuntimeMode `mapstructure:"mode,omitempty"`
}

func (cfg *RuntimeConfig) Validate() error {
	switch cfg.Name {
	// An empty name falls back to the default runtime (wazero) for
	// backward compatibility.
	case "", RuntimeNameWazero:
		if cfg.Mode != RuntimeModeInterpreter && cfg.Mode != RuntimeModeCompiled {
			return fmt.Errorf("invalid runtime mode: %s", cfg.Mode)
		}
	case RuntimeNameWasmEdge:
		// For WasmEdge, "interpreter" (default) executes the wasm module
		// directly and "compiled" AOT-compiles it at startup.
		if cfg.Mode != "" && cfg.Mode != RuntimeModeInterpreter && cfg.Mode != RuntimeModeCompiled {
			return fmt.Errorf("invalid runtime mode: %s", cfg.Mode)
		}
	default:
		return fmt.Errorf("invalid runtime name: %s", cfg.Name)
	}
	return nil
}

// Default sets the default values for the runtime configuration
// if they are not set.
func (cfg *RuntimeConfig) Default() {
	if cfg.Name == "" {
		cfg.Name = DefaultRuntimeConfig.Name
	}
	if cfg.Mode == "" && cfg.Name == RuntimeNameWazero {
		cfg.Mode = DefaultRuntimeConfig.Mode
	}
}

// DefaultRuntimeConfig is the default configuration for the WASM plugin runtime.
var DefaultRuntimeConfig = RuntimeConfig{
	Name: RuntimeNameWazero,
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
