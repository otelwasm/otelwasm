package wasmprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/stealthrocket/wasi-go"
	wasigo "github.com/stealthrocket/wasi-go/imports"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	guestExportMemory     = "memory"
	otelWasm              = "opentelemetry.io/wasm"
	currentTraces         = "currentTraces"
	currentMetrics        = "currentMetrics"
	currentLogs           = "currentLogs"
	setResultTraces       = "setResultTraces"
	setResultMetrics      = "setResultMetrics"
	setResultLogs         = "setResultLogs"
	getPluginConfig       = "getPluginConfig"
	setResultStatusReason = "setResultStatusReason"
)

type wasmProcessor struct {
	wasmProcessTraces  api.Function
	wasmProcessMetrics api.Function
	wasmProcessLogs    api.Function

	// pluginConfigJSON is the JSON representation of the plugin config.
	pluginConfigJSON []byte

	runtime wazero.Runtime
	sys     wasi.System
}

func newWasmProcessor(ctx context.Context, cfg *Config) (context.Context, *wasmProcessor, error) {
	if err := cfg.Validate(); err != nil {
		return ctx, nil, err
	}

	// TODO: We should invoke validate function defined in the guest at the iniitialization time
	// to check if the plugin config is valid. Currently it's checked every time when the process* function is called.

	f, err := os.Open(cfg.Path)
	if err != nil {
		return ctx, nil, err
	}
	defer f.Close()
	bytes, err := io.ReadAll(f)
	if err != nil {
		return ctx, nil, err
	}

	runtime, guest, err := prepareRuntime(ctx, bytes)
	if err != nil {
		return ctx, nil, err
	}

	// Instantiate WASI module (wasi_snapshot_preview1 and wasmedge socket extension)
	// TODO: Prepare own wasi_snapshot_preview1 package instead and remove wasi-go dependency in the future.
	var sys wasi.System
	ctx, sys, err = wasigo.NewBuilder().
		WithSocketsExtension("auto", guest).
		Instantiate(ctx, runtime)
	if err != nil {
		return ctx, nil, fmt.Errorf("wasm: error instantiating wasi module: %w", err)
	}

	if _, err := instantiateHostModule(ctx, runtime); err != nil {
		return ctx, nil, fmt.Errorf("wasm: error instantiating host module: %w", err)
	}

	config := wazero.NewModuleConfig().
		WithStartFunctions("_initialize"). // reactor module
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)
	mod, err := runtime.InstantiateModule(ctx, guest, config)
	if err != nil {
		return ctx, nil, fmt.Errorf("wasm: error instantiating guest: %w", err)
	}

	// TODO: Check the type of processors based on the exported functions becuase some processors might not support all telemetry types.
	processTraces := mod.ExportedFunction("processTraces")
	processMetrics := mod.ExportedFunction("processMetrics")
	processLogs := mod.ExportedFunction("processLogs")

	if processTraces == nil || processMetrics == nil || processLogs == nil {
		return ctx, nil, fmt.Errorf("wasm: guest doesn't export processTraces, processMetrics or processLogs")
	}

	// Convert the plugin config to JSON representation.
	pluginConfigJSON, err := json.Marshal(cfg.PluginConfig)
	if err != nil {
		return ctx, nil, fmt.Errorf("wasm: error marshalling plugin config: %w", err)
	}

	return ctx, &wasmProcessor{
		runtime:            runtime,
		wasmProcessTraces:  processTraces,
		wasmProcessMetrics: processMetrics,
		wasmProcessLogs:    processLogs,
		pluginConfigJSON:   pluginConfigJSON,
		sys:                sys,
	}, nil
}

func prepareRuntime(ctx context.Context, guestBin []byte) (runtime wazero.Runtime, guest wazero.CompiledModule, err error) {
	runtime = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig())

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
	currentTraces  ptrace.Traces
	currentMetrics pmetric.Metrics
	currentLogs    plog.Logs
	resultTraces   ptrace.Traces
	resultMetrics  pmetric.Metrics
	resultLogs     plog.Logs
	statusReason   string

	// pluginConfig is the plugin config in JSON representation passed to the guest.
	pluginConfigJSON []byte
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

func currentMetricsFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	metrics := paramsFromContext(ctx).currentMetrics
	stack[0] = uint64(marshalMetricsIfUnderLimit(mod.Memory(), metrics, buf, bufLimit))
}

func currentLogsFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	logs := paramsFromContext(ctx).currentLogs
	stack[0] = uint64(marshalLogsIfUnderLimit(mod.Memory(), logs, buf, bufLimit))
}

func getPluginConfigFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := bufLimit(stack[1])

	pluginConfig := paramsFromContext(ctx).pluginConfigJSON
	stack[0] = uint64(writeBytesIfUnderLimit(mod.Memory(), pluginConfig, buf, bufLimit))
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

func setResultMetricsFn(ctx context.Context, mod api.Module, stack []uint64) {
	// Read buffer pointer and size from the stack
	buf := uint32(stack[0])
	size := uint32(stack[1])

	// Read the serialized metrics from WASM memory
	metricsBytes, ok := mod.Memory().Read(buf, size)
	if !ok {
		panic("out of memory reading result metrics") // Bug: caller passed a length outside memory
	}

	// Unmarshal the metrics
	unmarshaler := pmetric.ProtoUnmarshaler{}
	metrics, err := unmarshaler.UnmarshalMetrics(metricsBytes)
	if err != nil {
		panic(err) // Bug: in unmarshaller
	}

	// Store the result metrics in context
	paramsFromContext(ctx).resultMetrics = metrics
}

func setResultLogsFn(ctx context.Context, mod api.Module, stack []uint64) {
	// Read buffer pointer and size from the stack
	buf := uint32(stack[0])
	size := uint32(stack[1])

	// Read the serialized logs from WASM memory
	logsBytes, ok := mod.Memory().Read(buf, size)
	if !ok {
		panic("out of memory reading result logs") // Bug: caller passed a length outside memory
	}

	// Unmarshal the logs
	unmarshaler := plog.ProtoUnmarshaler{}
	logs, err := unmarshaler.UnmarshalLogs(logsBytes)
	if err != nil {
		panic(err) // Bug: in unmarshaller
	}

	// Store the result logs in context
	paramsFromContext(ctx).resultLogs = logs
}

func setResultStatusReasonFn(ctx context.Context, mod api.Module, stack []uint64) {
	// Read buffer pointer and size from the stack
	buf := uint32(stack[0])
	size := uint32(stack[1])

	// Read the status reason string from WASM memory
	reasonBytes, ok := mod.Memory().Read(buf, size)
	if !ok {
		panic("out of memory reading status reason") // Bug: caller passed a length outside memory
	}

	// Store the status reason in context
	paramsFromContext(ctx).statusReason = string(reasonBytes)
}

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
		Instantiate(ctx)
}

func (wp *wasmProcessor) processTraces(
	ctx context.Context,
	td ptrace.Traces,
) (ptrace.Traces, error) {
	params := &stack{
		currentTraces:    td,
		pluginConfigJSON: wp.pluginConfigJSON,
	}
	ctx = context.WithValue(ctx, stackKey{}, params)

	// TODO: Use CallWithStack as it won't allocate call stack for every execution
	res, err := wp.wasmProcessTraces.Call(ctx)
	if err != nil {
		return td, err
	}

	statusCode := StatusCode(res[0])
	if statusCode != 0 {
		return td, fmt.Errorf("wasm: error processing traces: %s: %s", statusCode.String(), params.statusReason)
	}

	return params.resultTraces, nil
}

func (wp *wasmProcessor) processMetrics(
	ctx context.Context,
	md pmetric.Metrics,
) (pmetric.Metrics, error) {
	params := &stack{
		currentMetrics:   md,
		pluginConfigJSON: wp.pluginConfigJSON,
	}
	ctx = context.WithValue(ctx, stackKey{}, params)

	res, err := wp.wasmProcessMetrics.Call(ctx)
	if err != nil {
		return md, err
	}

	statusCode := StatusCode(res[0])
	if statusCode != 0 {
		return md, fmt.Errorf("wasm: error processing metrics: %s: %s", statusCode.String(), params.statusReason)
	}

	return params.resultMetrics, nil
}

func (wp *wasmProcessor) processLogs(
	ctx context.Context,
	ld plog.Logs,
) (plog.Logs, error) {
	params := &stack{
		currentLogs:      ld,
		pluginConfigJSON: wp.pluginConfigJSON,
	}
	ctx = context.WithValue(ctx, stackKey{}, params)

	res, err := wp.wasmProcessLogs.Call(ctx)
	if err != nil {
		return ld, err
	}

	statusCode := StatusCode(res[0])
	if statusCode != 0 {
		return ld, fmt.Errorf("wasm: error processing logs: %s: %s", statusCode.String(), params.statusReason)
	}

	return params.resultLogs, nil
}

func (wp *wasmProcessor) shutdown(ctx context.Context) error {
	if err := wp.sys.Close(ctx); err != nil {
		return fmt.Errorf("wasm: error closing system: %w", err)
	}
	if err := wp.runtime.Close(ctx); err != nil {
		return fmt.Errorf("wasm: error closing runtime: %w", err)
	}
	return nil
}
