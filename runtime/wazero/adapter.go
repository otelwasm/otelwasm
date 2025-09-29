package wazero

import (
	"context"
	"fmt"
	"os"

	"github.com/otelwasm/otelwasm/runtime"
	"github.com/stealthrocket/wasi-go"
	wasigo "github.com/stealthrocket/wasi-go/imports"
	"github.com/stealthrocket/wasi-go/imports/wasi_snapshot_preview1"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	// guestExportMemory is the name of the memory export in the guest module
	guestExportMemory = "memory"
	// wasmEdgeV2Extension is the WASI extension name
	wasmEdgeV2Extension = "wasmedgev2"
)

// wazeroRuntime implements runtime.Runtime using Wazero
type wazeroRuntime struct {
	runtime wazero.Runtime
	config  interface{} // TODO: Use proper config type after resolving circular dependency
}

// wazeroCompiledModule implements runtime.CompiledModule for Wazero
type wazeroCompiledModule struct {
	module  wazero.CompiledModule
	runtime wazero.Runtime
}

// wazeroModuleInstance implements runtime.ModuleInstance for Wazero
type wazeroModuleInstance struct {
	instance api.Module
}

// wazeroFunctionInstance implements runtime.FunctionInstance for Wazero
type wazeroFunctionInstance struct {
	function api.Function
}

// wazeroMemory implements runtime.Memory for Wazero
type wazeroMemory struct {
	memory api.Memory
}

// wazeroContext implements runtime.Context for Wazero
type wazeroContext struct {
	sys              wasi.System
	wasiP1HostModule *wasi_snapshot_preview1.Module
}

// Compile compiles the given Wasm binary into a CompiledModule
func (r *wazeroRuntime) Compile(ctx context.Context, binary []byte) (runtime.CompiledModule, error) {
	compiled, err := r.runtime.CompileModule(ctx, binary)
	if err != nil {
		return nil, fmt.Errorf("wazero compile error: %w", err)
	}

	// Validate memory export as per existing logic
	if _, ok := compiled.ExportedMemories()[guestExportMemory]; !ok {
		return nil, fmt.Errorf("wasm: guest doesn't export memory[%s]: %w", guestExportMemory, runtime.ErrMemoryExportNotFound)
	}

	return &wazeroCompiledModule{
		module:  compiled,
		runtime: r.runtime,
	}, nil
}

// InstantiateWithHost creates module instance with host functions and runtime-specific setup
func (r *wazeroRuntime) InstantiateWithHost(ctx context.Context, module runtime.CompiledModule, hostModule runtime.HostModule) (runtime.ModuleInstance, runtime.Context, error) {
	wazeroModule, ok := module.(*wazeroCompiledModule)
	if !ok {
		return nil, nil, fmt.Errorf("invalid module type for wazero runtime: %w", runtime.ErrInvalidConfiguration)
	}

	// Setup WASI
	var sys wasi.System
	ctx, sys, err := wasigo.NewBuilder().
		WithSocketsExtension(wasmEdgeV2Extension, wazeroModule.module).
		WithEnv(os.Environ()...).Instantiate(ctx, r.runtime)
	if err != nil {
		return nil, nil, fmt.Errorf("wasi instantiation failed: %w", err)
	}

	// Extract the wasi host module instance from the context as a workaround
	// to avoid panic when calling wasi functions with different context than the one used to instantiate the host module.
	wasiP1HostModule, ok := moduleInstanceFor[*wasi_snapshot_preview1.Module](ctx)
	if !ok {
		sys.Close(ctx)
		return nil, nil, fmt.Errorf("failed to retrieve wasi host module instance: %w", runtime.ErrInvalidConfiguration)
	}

	// Instantiate host module
	if _, err := r.instantiateHostModule(ctx, hostModule); err != nil {
		sys.Close(ctx)
		return nil, nil, fmt.Errorf("host module instantiation failed: %w", err)
	}

	// Instantiate the guest module
	config := wazero.NewModuleConfig().
		WithStartFunctions("_initialize"). // reactor module
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	instance, err := r.runtime.InstantiateModule(ctx, wazeroModule.module, config)
	if err != nil {
		sys.Close(ctx)
		return nil, nil, fmt.Errorf("guest module instantiation failed: %w", err)
	}

	runtimeCtx := &wazeroContext{
		sys:              sys,
		wasiP1HostModule: wasiP1HostModule,
	}

	return &wazeroModuleInstance{instance: instance}, runtimeCtx, nil
}

// Close closes the runtime and releases all resources
func (r *wazeroRuntime) Close(ctx context.Context) error {
	return r.runtime.Close(ctx)
}

// BuildOTelHostModule builds the OpenTelemetry host module for Wazero runtime
// This method provides access to the wazero runtime for host module building
func (r *wazeroRuntime) BuildOTelHostModule() (interface{}, error) {
	// Return the raw wazero runtime so wasmplugin can use it for instantiateHostModule
	return r.runtime, nil
}

// Close releases the resources associated with the compiled module
func (m *wazeroCompiledModule) Close(ctx context.Context) error {
	return m.module.Close(ctx)
}

// Function returns a handle to an exported function
func (m *wazeroModuleInstance) Function(name string) runtime.FunctionInstance {
	fn := m.instance.ExportedFunction(name)
	if fn == nil {
		return nil
	}
	return &wazeroFunctionInstance{function: fn}
}

// Memory returns the memory instance of the module
func (m *wazeroModuleInstance) Memory() runtime.Memory {
	memory := m.instance.Memory()
	if memory == nil {
		return nil
	}
	return &wazeroMemory{memory: memory}
}

// Close closes the instance and releases its resources
func (m *wazeroModuleInstance) Close(ctx context.Context) error {
	return m.instance.Close(ctx)
}

// Call executes the function with the given parameters
func (f *wazeroFunctionInstance) Call(ctx context.Context, params ...uint64) ([]uint64, error) {
	return f.function.Call(ctx, params...)
}

// Read reads 'size' bytes from the memory at 'offset'
func (mem *wazeroMemory) Read(offset uint32, size uint32) ([]byte, bool) {
	return mem.memory.Read(offset, size)
}

// Write writes 'data' to the memory at 'offset'
func (mem *wazeroMemory) Write(offset uint32, data []byte) bool {
	return mem.memory.Write(offset, data)
}

// Close releases runtime-specific resources
func (c *wazeroContext) Close(ctx context.Context) error {
	return c.sys.Close(ctx)
}

// WithRuntimeContext returns a context configured for runtime-specific operations
func (c *wazeroContext) WithRuntimeContext(ctx context.Context) context.Context {
	return withModuleInstance(ctx, c.wasiP1HostModule)
}

// instantiateHostModule creates and instantiates the host module with exported functions
func (r *wazeroRuntime) instantiateHostModule(ctx context.Context, hostModule runtime.HostModule) (api.Module, error) {
	builder := r.runtime.NewHostModuleBuilder("opentelemetry.io/wasm")

	// Register all host functions
	for _, hostFunc := range hostModule.GetFunctions() {
		// Get wazero-specific implementation
		wazeroImpl := hostFunc.Function.GetImplementation("wazero")
		if wazeroImpl == nil {
			return nil, fmt.Errorf("no wazero implementation for host function %s: %w", hostFunc.FunctionName, runtime.ErrHostFunctionNotFound)
		}

		// Cast to the expected function signature
		wazeroFunc, ok := wazeroImpl.(func(context.Context, api.Module, []uint64))
		if !ok {
			return nil, fmt.Errorf("invalid wazero function signature for %s: %w", hostFunc.FunctionName, runtime.ErrHostFunctionNotFound)
		}

		// Convert runtime.ValueType to api.ValueType
		paramTypes := make([]api.ValueType, len(hostFunc.ParamTypes))
		for i, vt := range hostFunc.ParamTypes {
			paramTypes[i] = convertValueType(vt)
		}

		resultTypes := make([]api.ValueType, len(hostFunc.ResultTypes))
		for i, vt := range hostFunc.ResultTypes {
			resultTypes[i] = convertValueType(vt)
		}

		builder = builder.NewFunctionBuilder().
			WithGoModuleFunction(api.GoModuleFunc(wazeroFunc), paramTypes, resultTypes).
			Export(hostFunc.FunctionName)
	}

	return builder.Instantiate(ctx)
}

// convertValueType converts runtime.ValueType to api.ValueType
func convertValueType(vt runtime.ValueType) api.ValueType {
	switch vt {
	case runtime.ValueTypeI32:
		return api.ValueTypeI32
	case runtime.ValueTypeI64:
		return api.ValueTypeI64
	case runtime.ValueTypeF32:
		return api.ValueTypeF32
	case runtime.ValueTypeF64:
		return api.ValueTypeF64
	default:
		return api.ValueTypeI32 // default fallback
	}
}

// moduleInstanceWrapper is a type wrapper for context values
type moduleInstanceWrapper[T any] struct{}

// moduleInstanceFor returns the module instance from the context that contains the internal
// state required for WASI host functions.
// NOTE: wasi-go returns context containing internal state when initializing the host module,
// and the same context is required when calling wasi functions exposed by wasi-go.
// This is a kind of workaround to avoid panic when calling
// wasi functions with different context than the one used to instantiate the host module.
func moduleInstanceFor[T wazergo.Module](ctx context.Context) (res T, ok bool) {
	res, ok = ctx.Value((*wazergo.ModuleInstance[T])(nil)).(T)
	return
}

// withModuleInstance returns a Go context inheriting from ctx and containing the
// state needed for module instantiated from wazero host module to properly bind
// their methods to their receiver (e.g. the module instance).
// NOTE: wasi-go returns context containing internal state when initializing the
// host module, and the same context is required when calling wasi functions
// exposed by wasi-go. This is a kind of workaround to avoid panic when calling
// wasi functions with different context than the one used to instantiate the host module.
func withModuleInstance[T wazergo.Module](ctx context.Context, instance T) context.Context {
	return context.WithValue(ctx, (*wazergo.ModuleInstance[T])(nil), instance)
}
