package wazero

import (
	"context"

	"github.com/tetratelabs/wazero"

	"github.com/otelwasm/otelwasm/runtime"
	"github.com/otelwasm/otelwasm/wasmplugin"
)

// newWazeroRuntime creates a new Wazero runtime instance
func newWazeroRuntime(config interface{}) (runtime.Runtime, error) {
	wazeroConfig, ok := config.(*wasmplugin.WazeroConfig)
	if !ok || wazeroConfig == nil {
		// Use default configuration
		wazeroConfig = &wasmplugin.WazeroConfig{
			Mode: wasmplugin.WazeroRuntimeModeInterpreter,
		}
	}

	// Create wazero runtime config based on mode
	var wrc wazero.RuntimeConfig
	switch wazeroConfig.Mode {
	case wasmplugin.WazeroRuntimeModeInterpreter, "":
		wrc = wazero.NewRuntimeConfigInterpreter()
	case wasmplugin.WazeroRuntimeModeCompiled:
		wrc = wazero.NewRuntimeConfigCompiler()
	default:
		wrc = wazero.NewRuntimeConfigInterpreter() // default fallback
	}

	wazeruntime := wazero.NewRuntimeWithConfig(context.Background(), wrc)

	return &wazeroRuntime{
		runtime: wazeruntime,
		config:  wazeroConfig,
	}, nil
}