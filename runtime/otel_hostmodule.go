package runtime

// This file contains OpenTelemetry-specific host module abstractions.
// The actual host function implementations remain in the wasmplugin package
// to maintain backward compatibility and reduce circular dependencies.

// HostModuleBuilderFunc is a function type that can build a host module
// for a specific runtime implementation
type HostModuleBuilderFunc func(runtime Runtime) (interface{}, error)

// TODO: OTelHostModuleBuilder will be implemented when circular dependency is resolved
