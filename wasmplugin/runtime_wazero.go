package wasmplugin

import (
	"context"
	"fmt"
	"os"

	"github.com/stealthrocket/wasi-go"
	wasigo "github.com/stealthrocket/wasi-go/imports"
	"github.com/stealthrocket/wasi-go/imports/wasi_snapshot_preview1"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// wazeroRuntime is the guestRuntime implementation backed by the wazero
// runtime. wazero is a pure-Go WebAssembly runtime and is always available.
type wazeroRuntime struct {
	// runtime is the WebAssembly runtime
	runtime wazero.Runtime

	// sys is the WASI system implementation
	sys wasi.System

	// module is the instantiated WASM module
	module api.Module

	// wasiP1HostModule is the host module instance initialized by wasi-go.
	// This instance holds necessary states for WASI host functions, which needs to be passed to context when calling the guest.
	// This is a workaround to avoid panic when calling wasi functions with different context than the one used to instantiate the host module.
	// TODO: Remove this if possible after replacing WASI implementation with our own.
	wasiP1HostModule *wasi_snapshot_preview1.Module
}

var _ guestRuntime = (*wazeroRuntime)(nil)

// stackKey is the key used to store the stack in the context
type stackKey struct{}

// paramsFromContext retrieves the Stack from the context
func paramsFromContext(ctx context.Context) *Stack {
	return ctx.Value(stackKey{}).(*Stack)
}

// createContextWithStack creates a new context with a Stack
func createContextWithStack(ctx context.Context, stack *Stack) context.Context {
	return context.WithValue(ctx, stackKey{}, stack)
}

// newWazeroRuntime initializes a wazero runtime and instantiates the guest module.
func newWazeroRuntime(ctx context.Context, guestBin []byte, rc RuntimeConfig) (*wazeroRuntime, error) {
	runtime, guest, err := prepareRuntime(ctx, guestBin, rc)
	if err != nil {
		return nil, err
	}

	// Instantiate WASI module (wasi_snapshot_preview1 and wasmedge socket extension)
	var sys wasi.System
	ctx, sys, err = wasigo.NewBuilder().
		WithSocketsExtension(wasmEdgeV2Extension, guest).
		WithEnv(os.Environ()...).Instantiate(ctx, runtime)
	if err != nil {
		return nil, fmt.Errorf("wasm: error instantiating wasi module: %w", err)
	}

	// Extract the wasi host module instance from the context as a workaround
	// to avoid panic when calling wasi functions with different context than the one used to instantiate the host module.
	wasiP1HostModule, ok := moduleInstanceFor[*wasi_snapshot_preview1.Module](ctx)
	if !ok {
		return nil, fmt.Errorf("wasm: error retrieving wasi host module instance")
	}

	if _, err := instantiateHostModule(ctx, runtime); err != nil {
		return nil, fmt.Errorf("wasm: error instantiating host module: %w", err)
	}

	config := wazero.NewModuleConfig().
		WithStartFunctions("_initialize"). // reactor module
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	mod, err := runtime.InstantiateModule(ctx, guest, config)
	if err != nil {
		return nil, fmt.Errorf("wasm: error instantiating guest: %w", err)
	}

	return &wazeroRuntime{
		runtime:          runtime,
		sys:              sys,
		module:           mod,
		wasiP1HostModule: wasiP1HostModule,
	}, nil
}

// prepareRuntime initializes a new WebAssembly runtime
func prepareRuntime(ctx context.Context, guestBin []byte, rc RuntimeConfig) (runtime wazero.Runtime, guest wazero.CompiledModule, err error) {
	// TODO: Switch to compiler backend after fixing the memory allocator issue in wazero
	var wrc wazero.RuntimeConfig
	switch rc.Mode {
	case RuntimeModeInterpreter:
		wrc = wazero.NewRuntimeConfigInterpreter()
	case RuntimeModeCompiled:
		// TODO: Add validation of supported platforms and architectures
		wrc = wazero.NewRuntimeConfigCompiler()
	default:
		return nil, nil, fmt.Errorf("wasm: invalid runtime mode: %s", rc.Mode)
	}
	runtime = wazero.NewRuntimeWithConfig(ctx, wrc)

	guest, err = compileGuest(ctx, runtime, guestBin)
	if err != nil {
		return nil, nil, err
	}

	return runtime, guest, nil
}

// compileGuest compiles the guest module
func compileGuest(ctx context.Context, runtime wazero.Runtime, guestBin []byte) (guest wazero.CompiledModule, err error) {
	if guest, err = runtime.CompileModule(ctx, guestBin); err != nil {
		err = fmt.Errorf("wasm: error compiling guest: %w", err)
	} else if _, ok := guest.ExportedMemories()[guestExportMemory]; !ok {
		// This section checks if the guest exports memory section.
		// As of WebAssembly Core Specification 2.0, there can be at most one memory.
		// https://webassembly.github.io/spec/core/syntax/modules.html#memories
		err = fmt.Errorf("wasm: guest doesn't export memory[%s]", guestExportMemory)
	}
	return
}

// HasFunction implements guestRuntime.
func (r *wazeroRuntime) HasFunction(name string) bool {
	return r.module.ExportedFunction(name) != nil
}

// CallFunction implements guestRuntime.
func (r *wazeroRuntime) CallFunction(ctx context.Context, name string, stack *Stack) ([]uint64, error) {
	ctx = createContextWithStack(ctx, stack)
	// Set the WASI host module instance in the context
	ctx = withModuleInstance(ctx, r.wasiP1HostModule)

	fn := r.module.ExportedFunction(name)
	if fn == nil {
		return nil, fmt.Errorf("wasm: function not found: %s", name)
	}

	return fn.Call(ctx)
}

// Close implements guestRuntime.
func (r *wazeroRuntime) Close(ctx context.Context) error {
	if err := r.sys.Close(ctx); err != nil {
		return fmt.Errorf("wasm: error closing system: %w", err)
	}
	if err := r.runtime.Close(ctx); err != nil {
		return fmt.Errorf("wasm: error closing runtime: %w", err)
	}
	return nil
}

// Host function implementations
func currentTracesFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := uint32(stack[1])

	stack[0] = uint64(hostCurrentTraces(paramsFromContext(ctx), mod.Memory(), buf, bufLimit))
}

func currentMetricsFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := uint32(stack[1])

	stack[0] = uint64(hostCurrentMetrics(paramsFromContext(ctx), mod.Memory(), buf, bufLimit))
}

func currentLogsFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := uint32(stack[1])

	stack[0] = uint64(hostCurrentLogs(paramsFromContext(ctx), mod.Memory(), buf, bufLimit))
}

func getPluginConfigFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := uint32(stack[1])

	stack[0] = uint64(hostGetPluginConfig(paramsFromContext(ctx), mod.Memory(), buf, bufLimit))
}

func getShutdownRequestedFn(ctx context.Context, mod api.Module, stack []uint64) {
	stack[0] = uint64(hostGetShutdownRequested(paramsFromContext(ctx)))
}

func setResultTracesFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	size := uint32(stack[1])

	hostSetResultTraces(paramsFromContext(ctx), mod.Memory(), buf, size)
}

func setResultMetricsFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	size := uint32(stack[1])

	hostSetResultMetrics(paramsFromContext(ctx), mod.Memory(), buf, size)
}

func setResultLogsFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	size := uint32(stack[1])

	hostSetResultLogs(paramsFromContext(ctx), mod.Memory(), buf, size)
}

func setResultStatusReasonFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	size := uint32(stack[1])

	hostSetResultStatusReason(paramsFromContext(ctx), mod.Memory(), buf, size)
}

// instantiateHostModule creates and instantiates the host module with exported functions
func instantiateHostModule(ctx context.Context, runtime wazero.Runtime) (api.Module, error) {
	return runtime.NewHostModuleBuilder(otelWasm).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(currentTracesFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		WithParameterNames("buf", "buf_limit").Export(currentTraces).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(currentMetricsFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		WithParameterNames("buf", "buf_limit").Export(currentMetrics).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(currentLogsFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		WithParameterNames("buf", "buf_limit").Export(currentLogs).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultTracesFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(setResultTraces).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultMetricsFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(setResultMetrics).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultLogsFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(setResultLogs).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(getPluginConfigFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		WithParameterNames("buf", "buf_limit").Export(getPluginConfig).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultStatusReasonFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(setResultStatusReason).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(getShutdownRequestedFn), []api.ValueType{}, []api.ValueType{api.ValueTypeI32}).
		Export(getShutdownRequested).
		Instantiate(ctx)
}

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
