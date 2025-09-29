package wazero

import (
	"context"

	"github.com/otelwasm/otelwasm/runtime"
	"github.com/tetratelabs/wazero"
)

// newWazeroRuntime creates a new Wazero runtime instance
func newWazeroRuntime(config interface{}) (runtime.Runtime, error) {
	// TODO: Parse config properly after resolving circular dependency
	// For now, use default interpreter mode
	wrc := wazero.NewRuntimeConfigInterpreter()

	wazeruntime := wazero.NewRuntimeWithConfig(context.Background(), wrc)

	return &wazeroRuntime{
		runtime: wazeruntime,
		config:  config,
	}, nil
}
