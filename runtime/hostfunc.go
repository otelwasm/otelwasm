package runtime

import (
	"context"

	"github.com/tetratelabs/wazero/api"
)

// HostFunctionImpl represents a runtime-specific implementation of a host function
type HostFunctionImpl interface {
	// GetImplementation returns the runtime-specific implementation
	// The runtimeType parameter specifies which runtime implementation to return
	GetImplementation(runtimeType string) interface{}
}

// WazeroHostFunction wraps a Wazero-specific host function implementation
type WazeroHostFunction struct {
	Function func(context.Context, api.Module, []uint64)
}

// GetImplementation returns the Wazero implementation
func (w *WazeroHostFunction) GetImplementation(runtimeType string) interface{} {
	switch runtimeType {
	case "wazero":
		return w.Function
	default:
		return nil
	}
}

// HostFunctionDefinition defines a host function with its signature and implementations
type HostFunctionDefinition struct {
	FunctionName string
	ParamTypes   []ValueType
	ResultTypes  []ValueType
	Function     HostFunctionImpl
}

// HostModule represents a collection of host functions that can be instantiated in any runtime
type HostModule struct {
	Name      string
	Functions []HostFunctionDefinition
}

// NewHostModule creates a new host module with the given name
func NewHostModule(name string) *HostModule {
	return &HostModule{
		Name:      name,
		Functions: make([]HostFunctionDefinition, 0),
	}
}

// AddFunction adds a host function to the module
func (hm *HostModule) AddFunction(name string, paramTypes, resultTypes []ValueType, impl HostFunctionImpl) {
	hm.Functions = append(hm.Functions, HostFunctionDefinition{
		FunctionName: name,
		ParamTypes:   paramTypes,
		ResultTypes:  resultTypes,
		Function:     impl,
	})
}

// Functions returns all host function definitions
func (hm *HostModule) Functions() []HostFunctionDefinition {
	return hm.Functions
}