package runtime

// HostFunction represents a single host function definition
type HostFunction struct {
	ModuleName   string
	FunctionName string
	Function     interface{} // Runtime-specific function implementation
	ParamTypes   []ValueType
	ResultTypes  []ValueType
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
type HostFunctionImpl struct {
	Name string
}

// GetImplementation returns the runtime-specific implementation
// This method will be called by runtime adapters to get their specific function implementation
func (h *HostFunctionImpl) GetImplementation(runtimeType string) interface{} {
	return getHostFunction(runtimeType, h.Name)
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