package wasmprocessor

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	guestExportMemory = "memory"
	otelWasm          = "opentelemetry.io/wasm"
	currentTraces     = "currentTraces"
	setResultTraces   = "setResultTraces"
)

type wasmProcessor struct {
	wasmProcessTraces api.Function

	runtime wazero.Runtime
}

func newWasmProcessor(ctx context.Context, cfg *Config) (*wasmProcessor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	f, err := os.Open(cfg.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	runtime, guest, err := prepareRuntime(ctx, bytes)
	if err != nil {
		return nil, err
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

	processTraces := mod.ExportedFunction("processTraces")
	if processTraces == nil {
		return nil, fmt.Errorf("wasm: error getting processTraces function")
	}

	return &wasmProcessor{
		runtime:           runtime,
		wasmProcessTraces: processTraces,
	}, nil
}

func prepareRuntime(ctx context.Context, guestBin []byte) (runtime wazero.Runtime, guest wazero.CompiledModule, err error) {
	runtime = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig())

	_, err = wasi_snapshot_preview1.Instantiate(ctx, runtime)
	if err != nil {
		return nil, nil, err
	}

	guest, err = compileGuest(ctx, runtime, guestBin)
	if err != nil {
		return nil, nil, err
	}

	return runtime, guest, nil
}

func compileGuest(ctx context.Context, runtime wazero.Runtime, guestBin []byte) (guest wazero.CompiledModule, err error) {
	if guest, err = runtime.CompileModule(ctx, guestBin); err != nil {
		err = fmt.Errorf("wasm: error compiling guest: %w", err)
	} else if _, ok := guest.ExportedMemories()[guestExportMemory]; !ok {
		// This section checkes if the guest exports memory section.
		// As of WebAssembly Core Specification 2.0, there can be at most one memory.
		// https://webassembly.github.io/spec/core/syntax/modules.html#memories
		err = fmt.Errorf("wasm: guest doesn't export memory[%s]", guestExportMemory)
	}
	return
}

type stackKey struct{}

type stack struct {
	currentTraces ptrace.Traces
	resultTraces  ptrace.Traces
}

func paramsFromContext(ctx context.Context) *stack {
	return ctx.Value(stackKey{}).(*stack)
}

func currentTracesFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	traces := paramsFromContext(ctx).currentTraces
	stack[0] = uint64(marshalTraceIfUnderLimit(mod.Memory(), traces, buf, bufLimit))
}

func setResultTracesFn(ctx context.Context, mod api.Module, stack []uint64) {
	// Read buffer pointer and size from the stack
	buf := uint32(stack[0])
	size := uint32(stack[1])

	// Read the serialized traces from WASM memory
	tracesBytes, ok := mod.Memory().Read(buf, size)
	if !ok {
		panic("out of memory reading result traces") // Bug: caller passed a length outside memory
	}

	// Unmarshal the traces
	unmarshaler := ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(tracesBytes)
	if err != nil {
		panic(err) // Bug: in unmarshaller
	}

	// Store the result traces in context
	paramsFromContext(ctx).resultTraces = traces
}

func instantiateHostModule(ctx context.Context, runtime wazero.Runtime) (api.Module, error) {
	return runtime.NewHostModuleBuilder(otelWasm).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(currentTracesFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		WithParameterNames("buf", "buf_limit").Export(currentTraces).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultTracesFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(setResultTraces).
		Instantiate(ctx)
}

func (wp *wasmProcessor) processTraces(
	ctx context.Context,
	td ptrace.Traces,
) (ptrace.Traces, error) {
	params := &stack{
		currentTraces: td,
	}
	ctx = context.WithValue(ctx, stackKey{}, params)

	// TODO: Use CallWithStack as it won't allocate call stack for every execution
	res, err := wp.wasmProcessTraces.Call(ctx)
	if err != nil {
		return td, err
	}

	statusCode := int32(res[0])
	if statusCode != 0 {
		return td, fmt.Errorf("wasm: error processing traces: %d", statusCode)
	}

	return params.resultTraces, nil
}
