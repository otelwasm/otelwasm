// Package runtime provides an abstraction layer for WebAssembly runtime engines.
package runtime

import "context"

// Runtime represents a Wasm runtime engine
type Runtime interface {
	// Compile compiles the given Wasm binary into a CompiledModule
	Compile(ctx context.Context, binary []byte) (CompiledModule, error)
	// InstantiateWithHost creates module instance with host functions and runtime-specific setup
	InstantiateWithHost(ctx context.Context, module CompiledModule, hostModule HostModule) (ModuleInstance, Context, error)
	// Close closes the runtime and releases all resources
	Close(ctx context.Context) error
}

// CompiledModule represents a compiled Wasm module, ready for instantiation
type CompiledModule interface {
	// Close releases the resources associated with the compiled module
	Close(ctx context.Context) error
}

// ModuleInstance represents an instantiated Wasm module
type ModuleInstance interface {
	// Function returns a handle to an exported function
	// Returns nil if the function is not found
	Function(name string) FunctionInstance
	// Memory returns the memory instance of the module
	// Returns nil if the module does not export memory
	Memory() Memory
	// Close closes the instance and releases its resources
	Close(ctx context.Context) error
}

// FunctionInstance represents an exported function from a Wasm module
type FunctionInstance interface {
	// Call executes the function with the given parameters
	Call(ctx context.Context, params ...uint64) ([]uint64, error)
}

// Memory represents the linear memory of a Wasm module instance
type Memory interface {
	// Read reads 'size' bytes from the memory at 'offset'
	Read(offset uint32, size uint32) ([]byte, bool)
	// Write writes 'data' to the memory at 'offset'
	Write(offset uint32, data []byte) bool
}

// Context holds runtime-specific state (WASI, host modules, etc.)
// This is opaque to WasmPlugin and managed entirely by runtime adapters
type Context interface {
	// Close releases runtime-specific resources
	Close(ctx context.Context) error
}

// HostModule defines host functions to be made available to WASM modules
type HostModule interface {
	// Functions returns the list of host functions to register
	Functions() []HostFunction
}