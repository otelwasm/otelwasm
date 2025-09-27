package runtime

import (
	"context"
	"fmt"

	"github.com/otelwasm/otelwasm/wasmplugin"
)

// Factory is a function that creates a new Runtime
type Factory func(config interface{}) (Runtime, error)

var runtimeFactories = make(map[string]Factory)

// Register registers a runtime factory
func Register(name string, factory Factory) {
	if _, exists := runtimeFactories[name]; exists {
		panic(fmt.Sprintf("runtime %s already registered", name))
	}
	runtimeFactories[name] = factory
}

// New creates a new Runtime based on the config
func New(ctx context.Context, config *wasmplugin.RuntimeConfig) (Runtime, error) {
	// Default to wazero if not specified
	runtimeType := config.Type
	if runtimeType == "" {
		runtimeType = wasmplugin.RuntimeTypeWazero
	}

	factory, ok := runtimeFactories[runtimeType]
	if !ok {
		return nil, fmt.Errorf("unknown runtime type: %s: %w", runtimeType, ErrRuntimeNotFound)
	}

	// Extract runtime-specific configuration
	var specificConfig interface{}
	switch runtimeType {
	case wasmplugin.RuntimeTypeWazero:
		specificConfig = config.Wazero
	case wasmplugin.RuntimeTypeWasmtime:
		specificConfig = config.Wasmtime
	default:
		specificConfig = config.Remaining[runtimeType]
	}

	return factory(specificConfig)
}

// List returns all registered runtime types
func List() []string {
	types := make([]string, 0, len(runtimeFactories))
	for t := range runtimeFactories {
		types = append(types, t)
	}
	return types
}