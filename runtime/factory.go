package runtime

import (
	"fmt"
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

// NewRuntime creates a new Runtime by name and config
func NewRuntime(runtimeType string, config interface{}) (Runtime, error) {
	// Default to wazero if not specified
	if runtimeType == "" {
		runtimeType = "wazero"
	}

	factory, ok := runtimeFactories[runtimeType]
	if !ok {
		return nil, fmt.Errorf("unknown runtime type: %s: %w", runtimeType, ErrRuntimeNotFound)
	}

	return factory(config)
}

// List returns all registered runtime types
func List() []string {
	types := make([]string, 0, len(runtimeFactories))
	for t := range runtimeFactories {
		types = append(types, t)
	}
	return types
}
