package runtime

import (
	"context"

	"github.com/tetratelabs/wazero/api"
)

// HostFunction represents a single host function definition
type HostFunction struct {
	ModuleName   string
	FunctionName string
	Function     interface{} // Runtime-specific function implementation
	ParamTypes   []ValueType
	ResultTypes  []ValueType
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

// ValueType represents WASM value types
type ValueType int

const (
	ValueTypeI32 ValueType = iota
	ValueTypeI64
	ValueTypeF32
	ValueTypeF64
)

// HostFunctionImpl provides a runtime-agnostic representation of host functions
// The actual implementation will be provided by the runtime adapter
type HostFunctionImpl interface {
	// GetImplementation returns the runtime-specific implementation
	// The runtimeType parameter specifies which runtime implementation to return
	GetImplementation(runtimeType string) interface{}
}

// hostFunctionRegistry holds runtime-specific host function implementations
var hostFunctionRegistry = make(map[string]map[string]interface{})

// RegisterHostFunction registers a runtime-specific host function implementation
func RegisterHostFunction(runtimeType, functionName string, implementation interface{}) {
	if hostFunctionRegistry[runtimeType] == nil {
		hostFunctionRegistry[runtimeType] = make(map[string]interface{})
	}
	hostFunctionRegistry[runtimeType][functionName] = implementation
}

// getHostFunction returns the runtime-specific host function implementation
func getHostFunction(runtimeType, functionName string) interface{} {
	if runtimeFunctions, ok := hostFunctionRegistry[runtimeType]; ok {
		return runtimeFunctions[functionName]
	}
	return nil
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

// GetFunctions returns all host function definitions
func (hm *HostModule) GetFunctions() []HostFunctionDefinition {
	return hm.Functions
}
