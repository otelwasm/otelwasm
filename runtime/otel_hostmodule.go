package runtime

// This file contains OpenTelemetry-specific host module abstractions.
// The actual host function implementations remain in the wasmplugin package
// to maintain backward compatibility and reduce circular dependencies.

// HostModuleBuilderFunc is a function type that can build a host module
// for a specific runtime implementation
type HostModuleBuilderFunc func(runtime Runtime) (interface{}, error)

// OTelHostModuleBuilder builds the OpenTelemetry host module for the given runtime
func OTelHostModuleBuilder(rt Runtime) (interface{}, error) {
	switch v := rt.(type) {
	case *wazeroRuntime:
		// For Wazero runtime, delegate to wasmplugin's instantiateHostModule
		// This maintains existing functionality while allowing abstraction
		return v.BuildOTelHostModule()
	default:
		// Future runtime implementations will add their own host module builders here
		return nil, ErrRuntimeNotSupported
	}
}