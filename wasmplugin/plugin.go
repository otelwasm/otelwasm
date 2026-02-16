// Package wasmplugin provides common functionality for WebAssembly-based OpenTelemetry components.
package wasmplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/stealthrocket/wasi-go"
	wasigo "github.com/stealthrocket/wasi-go/imports"
	"github.com/stealthrocket/wasi-go/imports/wasi_snapshot_preview1"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	// guestExportMemory is the name of the memory export in the guest module
	guestExportMemory = "memory"

	// otelWasm is the name of the host module
	otelWasm = "opentelemetry.io/wasm"

	// Host function exports
	setResultTraces      = "set_result_traces"
	setResultMetrics     = "set_result_metrics"
	setResultLogs        = "set_result_logs"
	getPluginConfig      = "get_plugin_config"
	setStatusReason      = "set_status_reason"
	getShutdownRequested = "get_shutdown_requested"

	// Legacy host export names used by pre-migration modules.
	legacySetResultTraces       = "setResultTraces"
	legacySetResultMetrics      = "setResultMetrics"
	legacySetResultLogs         = "setResultLogs"
	legacyGetPluginConfig       = "getPluginConfig"
	legacySetResultStatusReason = "setResultStatusReason"
	legacyGetShutdownRequested  = "getShutdownRequested"

	// Guest function
	getSupportedTelemetry       = "get_supported_telemetry"
	legacyGetSupportedTelemetry = "getSupportedTelemetry"
	memoryAllocateFunction      = "otelwasm_memory_allocate"
	consumeTracesFunction       = "otelwasm_consume_traces"
	consumeMetricsFunction      = "otelwasm_consume_metrics"
	consumeLogsFunction         = "otelwasm_consume_logs"

	// WASI extension name
	wasmEdgeV2Extension = "wasmedgev2"
)

var builtInGuestFunctions = map[string][]string{
	getSupportedTelemetry: {getSupportedTelemetry, legacyGetSupportedTelemetry},
}

var abiV1RequiredFunctions = map[string]struct{}{
	"otelwasm_consume_traces":         {},
	"otelwasm_consume_metrics":        {},
	"otelwasm_consume_logs":           {},
	"start":                           {},
	"shutdown":                        {},
	"otelwasm_start_traces_receiver":  {},
	"otelwasm_start_metrics_receiver": {},
	"otelwasm_start_logs_receiver":    {},
	// Legacy naming still used by non-migrated call sites.
	"startTracesReceiver":  {},
	"startMetricsReceiver": {},
	"startLogsReceiver":    {},
}

type telemetryType uint32

const (
	telemetryTypeMetrics telemetryType = 1 << iota
	telemetryTypeLogs
	telemetryTypeTraces
)

// StatusCode represents the result status code from WASM function calls
type StatusCode uint32

// String returns the string representation of the status code
func (s StatusCode) String() string {
	switch s {
	case 0:
		return "OK"
	case 1:
		return "ERROR"
	case 2:
		return "INVALID_ARGUMENT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

// WasmPlugin represents a WebAssembly plugin for OpenTelemetry components
type WasmPlugin struct {
	consumeMu sync.Mutex

	// Runtime is the WebAssembly runtime
	Runtime wazero.Runtime

	// System is the WASI system implementation
	Sys wasi.System

	// Module is the instantiated WASM module
	Module api.Module

	// ABIVersion is the detected ABI version implemented by the module.
	ABIVersion ABIVersion

	// PluginConfigJSON is the JSON representation of the plugin config
	PluginConfigJSON []byte

	// Exported functions from the WASM module
	ExportedFunctions map[string]api.Function

	// wasiP1HostModule is the host module instance initialized by wasi-go.
	// This instance holds necessary states for WASI host functions, which needs to be passed to context when calling the guest.
	// This is a workaround to avoid panic when calling wasi functions with different context than the one used to instantiate the host module.
	// TODO: Remove this if possible after replacing WASI implementation with our own.
	wasiP1HostModule *wasi_snapshot_preview1.Module
}

// stackKey is the key used to store the stack in the context
type stackKey struct{}

// Stack holds the data being passed between the host and the guest
type Stack struct {
	ResultTraces      ptrace.Traces
	ResultMetrics     pmetric.Metrics
	ResultLogs        plog.Logs
	StatusReason      string
	RequestedShutdown atomic.Bool

	OnResultMetricsChange func(pmetric.Metrics)
	OnResultLogsChange    func(plog.Logs)
	OnResultTracesChange  func(ptrace.Traces)

	// PluginConfigJSON is the plugin config in JSON representation passed to the guest
	PluginConfigJSON []byte
}

// paramsFromContext retrieves the Stack from the context
func paramsFromContext(ctx context.Context) *Stack {
	return ctx.Value(stackKey{}).(*Stack)
}

// NewWasmPlugin creates a new WasmPlugin instance
func NewWasmPlugin(ctx context.Context, cfg *Config, requiredFunctions []string) (*WasmPlugin, error) {
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

	runtime, guest, err := prepareRuntime(ctx, bytes, cfg.RuntimeConfig)
	if err != nil {
		return nil, err
	}
	if requiresABIV1(requiredFunctions) && guest.ExportedFunctions()[abiVersionV1MarkerExport] == nil {
		return nil, fmt.Errorf("wasm: %s is not exported: %w", abiVersionV1MarkerExport, ErrABIVersionMarkerNotExported)
	}

	// Instantiate WASI module (wasi_snapshot_preview1 and wasmedge socket extension)
	var sys wasi.System
	ctx, sys, err = wasigo.NewBuilder().
		WithSocketsExtension(wasmEdgeV2Extension, guest).
		WithStdio(int(os.Stdin.Fd()), int(os.Stdout.Fd()), int(os.Stderr.Fd())).
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
	abiVersion := detectABIVersion(mod)
	if requiresABIV1(requiredFunctions) && abiVersion != ABIV1 {
		return nil, fmt.Errorf("wasm: %s is not exported: %w", abiVersionV1MarkerExport, ErrABIVersionMarkerNotExported)
	}

	// Check if all required functions are exported
	exportedFunctions := make(map[string]api.Function)
	for _, funcName := range requiredFunctions {
		fn := mod.ExportedFunction(funcName)
		if fn == nil {
			return nil, fmt.Errorf("wasm: %s is not exported: %w", funcName, ErrRequiredFunctionNotExported)
		}
		exportedFunctions[funcName] = fn
	}

	// Check if all built-in guest functions are exported
	for canonicalName, aliases := range builtInGuestFunctions {
		var fn api.Function
		for _, name := range aliases {
			fn = mod.ExportedFunction(name)
			if fn != nil {
				break
			}
		}
		if fn == nil {
			return nil, fmt.Errorf("wasm: %s is not exported: %w", canonicalName, ErrRequiredFunctionNotExported)
		}
		exportedFunctions[canonicalName] = fn
	}

	// Convert the plugin config to JSON representation
	pluginConfigJSON, err := json.Marshal(cfg.PluginConfig)
	if err != nil {
		return nil, fmt.Errorf("wasm: error marshalling plugin config: %w", err)
	}

	plugin := &WasmPlugin{
		Runtime:           runtime,
		Sys:               sys,
		Module:            mod,
		ABIVersion:        abiVersion,
		PluginConfigJSON:  pluginConfigJSON,
		ExportedFunctions: exportedFunctions,
		wasiP1HostModule:  wasiP1HostModule,
	}

	return plugin, nil
}

func requiresABIV1(requiredFunctions []string) bool {
	for _, name := range requiredFunctions {
		if _, ok := abiV1RequiredFunctions[name]; ok {
			return true
		}
	}
	return false
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

// createContextWithStack creates a new context with a Stack
func createContextWithStack(ctx context.Context, stack *Stack) context.Context {
	return context.WithValue(ctx, stackKey{}, stack)
}

// ProcessFunctionCall executes a WASM function and handles stack management
func (p *WasmPlugin) ProcessFunctionCall(ctx context.Context, functionName string, stack *Stack) ([]uint64, error) {
	fn, ok := p.ExportedFunctions[functionName]
	if !ok {
		return nil, fmt.Errorf("wasm: function not found: %s", functionName)
	}

	return p.callFunction(ctx, fn, stack)
}

func (p *WasmPlugin) callFunction(ctx context.Context, fn api.Function, stack *Stack, params ...uint64) ([]uint64, error) {
	if stack == nil {
		stack = &Stack{}
	}
	ctx = createContextWithStack(ctx, stack)
	// Set the WASI host module instance in the context.
	ctx = withModuleInstance(ctx, p.wasiP1HostModule)
	return fn.Call(ctx, params...)
}

func (p *WasmPlugin) ConsumeTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	p.consumeMu.Lock()
	defer p.consumeMu.Unlock()

	marshaler := ptrace.ProtoMarshaler{}
	payload, err := marshaler.MarshalTraces(td)
	if err != nil {
		return td, fmt.Errorf("wasm: failed to marshal traces: %w", err)
	}

	stack := &Stack{PluginConfigJSON: p.PluginConfigJSON}
	var dataPtr uint64
	dataSize := uint64(len(payload))

	if len(payload) > 0 {
		allocFn := p.Module.ExportedFunction(memoryAllocateFunction)
		if allocFn == nil {
			return td, fmt.Errorf("wasm: %s is not exported", memoryAllocateFunction)
		}

		allocRes, allocErr := p.callFunction(ctx, allocFn, nil, dataSize)
		if allocErr != nil {
			return td, fmt.Errorf("wasm: failed to call %s: %w", memoryAllocateFunction, allocErr)
		}
		if len(allocRes) == 0 || allocRes[0] == 0 {
			return td, fmt.Errorf("wasm: %s returned null for %d bytes", memoryAllocateFunction, len(payload))
		}

		dataPtr = allocRes[0]
		if !p.Module.Memory().Write(uint32(dataPtr), payload) {
			return td, fmt.Errorf("wasm: failed to write traces payload to guest memory")
		}
	}

	consumeFn, ok := p.ExportedFunctions[consumeTracesFunction]
	if !ok {
		consumeFn = p.Module.ExportedFunction(consumeTracesFunction)
		if consumeFn == nil {
			return td, fmt.Errorf("wasm: %s is not exported", consumeTracesFunction)
		}
	}

	result, err := p.callFunction(ctx, consumeFn, stack, dataPtr, dataSize)
	if err != nil {
		return td, err
	}
	if len(result) == 0 {
		return td, fmt.Errorf("wasm: %s returned no status code", consumeTracesFunction)
	}
	statusCode := StatusCode(result[0])
	if statusCode != 0 {
		return td, fmt.Errorf("wasm: error processing traces: %s: %s", statusCode.String(), stack.StatusReason)
	}

	if stack.ResultTraces != (ptrace.Traces{}) {
		return stack.ResultTraces, nil
	}
	return td, nil
}

func (p *WasmPlugin) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	p.consumeMu.Lock()
	defer p.consumeMu.Unlock()

	marshaler := pmetric.ProtoMarshaler{}
	payload, err := marshaler.MarshalMetrics(md)
	if err != nil {
		return md, fmt.Errorf("wasm: failed to marshal metrics: %w", err)
	}

	stack := &Stack{PluginConfigJSON: p.PluginConfigJSON}
	var dataPtr uint64
	dataSize := uint64(len(payload))

	if len(payload) > 0 {
		allocFn := p.Module.ExportedFunction(memoryAllocateFunction)
		if allocFn == nil {
			return md, fmt.Errorf("wasm: %s is not exported", memoryAllocateFunction)
		}

		allocRes, allocErr := p.callFunction(ctx, allocFn, nil, dataSize)
		if allocErr != nil {
			return md, fmt.Errorf("wasm: failed to call %s: %w", memoryAllocateFunction, allocErr)
		}
		if len(allocRes) == 0 || allocRes[0] == 0 {
			return md, fmt.Errorf("wasm: %s returned null for %d bytes", memoryAllocateFunction, len(payload))
		}

		dataPtr = allocRes[0]
		if !p.Module.Memory().Write(uint32(dataPtr), payload) {
			return md, fmt.Errorf("wasm: failed to write metrics payload to guest memory")
		}
	}

	consumeFn, ok := p.ExportedFunctions[consumeMetricsFunction]
	if !ok {
		consumeFn = p.Module.ExportedFunction(consumeMetricsFunction)
		if consumeFn == nil {
			return md, fmt.Errorf("wasm: %s is not exported", consumeMetricsFunction)
		}
	}

	result, err := p.callFunction(ctx, consumeFn, stack, dataPtr, dataSize)
	if err != nil {
		return md, err
	}
	if len(result) == 0 {
		return md, fmt.Errorf("wasm: %s returned no status code", consumeMetricsFunction)
	}
	statusCode := StatusCode(result[0])
	if statusCode != 0 {
		return md, fmt.Errorf("wasm: error processing metrics: %s: %s", statusCode.String(), stack.StatusReason)
	}

	if stack.ResultMetrics != (pmetric.Metrics{}) {
		return stack.ResultMetrics, nil
	}
	return md, nil
}

func (p *WasmPlugin) ConsumeLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	p.consumeMu.Lock()
	defer p.consumeMu.Unlock()

	marshaler := plog.ProtoMarshaler{}
	payload, err := marshaler.MarshalLogs(ld)
	if err != nil {
		return ld, fmt.Errorf("wasm: failed to marshal logs: %w", err)
	}

	stack := &Stack{PluginConfigJSON: p.PluginConfigJSON}
	var dataPtr uint64
	dataSize := uint64(len(payload))

	if len(payload) > 0 {
		allocFn := p.Module.ExportedFunction(memoryAllocateFunction)
		if allocFn == nil {
			return ld, fmt.Errorf("wasm: %s is not exported", memoryAllocateFunction)
		}

		allocRes, allocErr := p.callFunction(ctx, allocFn, nil, dataSize)
		if allocErr != nil {
			return ld, fmt.Errorf("wasm: failed to call %s: %w", memoryAllocateFunction, allocErr)
		}
		if len(allocRes) == 0 || allocRes[0] == 0 {
			return ld, fmt.Errorf("wasm: %s returned null for %d bytes", memoryAllocateFunction, len(payload))
		}

		dataPtr = allocRes[0]
		if !p.Module.Memory().Write(uint32(dataPtr), payload) {
			return ld, fmt.Errorf("wasm: failed to write logs payload to guest memory")
		}
	}

	consumeFn, ok := p.ExportedFunctions[consumeLogsFunction]
	if !ok {
		consumeFn = p.Module.ExportedFunction(consumeLogsFunction)
		if consumeFn == nil {
			return ld, fmt.Errorf("wasm: %s is not exported", consumeLogsFunction)
		}
	}

	result, err := p.callFunction(ctx, consumeFn, stack, dataPtr, dataSize)
	if err != nil {
		return ld, err
	}
	if len(result) == 0 {
		return ld, fmt.Errorf("wasm: %s returned no status code", consumeLogsFunction)
	}
	statusCode := StatusCode(result[0])
	if statusCode != 0 {
		return ld, fmt.Errorf("wasm: error processing logs: %s: %s", statusCode.String(), stack.StatusReason)
	}

	if stack.ResultLogs != (plog.Logs{}) {
		return stack.ResultLogs, nil
	}
	return ld, nil
}

func (p *WasmPlugin) supportedTelemetryTypes(ctx context.Context) (telemetryType, error) {
	// TODO: Cache the result of this function to avoid calling it multiple times

	res, err := p.ProcessFunctionCall(ctx, getSupportedTelemetry, &Stack{})
	if err != nil {
		return 0, fmt.Errorf("wasm: failed to get supported telemetry types: %w", err)
	}

	if len(res) == 0 {
		return 0, fmt.Errorf("wasm: no supported telemetry types returned")
	}

	return telemetryType(res[0]), nil
}

func (p *WasmPlugin) IsMetricsSupported(ctx context.Context) (bool, error) {
	telemetryTypes, err := p.supportedTelemetryTypes(ctx)
	if err != nil {
		return false, err
	}
	return telemetryTypes&telemetryTypeMetrics != 0, nil
}

func (p *WasmPlugin) IsLogsSupported(ctx context.Context) (bool, error) {
	telemetryTypes, err := p.supportedTelemetryTypes(ctx)
	if err != nil {
		return false, err
	}
	return telemetryTypes&telemetryTypeLogs != 0, nil
}

func (p *WasmPlugin) IsTracesSupported(ctx context.Context) (bool, error) {
	telemetryTypes, err := p.supportedTelemetryTypes(ctx)
	if err != nil {
		return false, err
	}
	return telemetryTypes&telemetryTypeTraces != 0, nil
}

// Shutdown closes the WASM runtime and system
func (p *WasmPlugin) Shutdown(ctx context.Context) error {
	if err := p.Sys.Close(ctx); err != nil {
		return fmt.Errorf("wasm: error closing system: %w", err)
	}
	if err := p.Runtime.Close(ctx); err != nil {
		return fmt.Errorf("wasm: error closing runtime: %w", err)
	}
	return nil
}

func getPluginConfigFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := uint32(stack[1])

	pluginConfig := paramsFromContext(ctx).PluginConfigJSON
	stack[0] = uint64(writeBytesIfUnderLimit(mod.Memory(), pluginConfig, buf, bufLimit))
}

func getShutdownRequestedFn(ctx context.Context, mod api.Module, stack []uint64) {
	// Read the shutdown requested flag from the stack
	shutdownRequested := paramsFromContext(ctx).RequestedShutdown.Load()

	// Write the shutdown requested flag to the stack
	if shutdownRequested {
		stack[0] = 1
	} else {
		stack[0] = 0
	}
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
	paramsFromContext(ctx).ResultTraces = traces
	onResultTracesChange := paramsFromContext(ctx).OnResultTracesChange
	if onResultTracesChange != nil {
		onResultTracesChange(traces)
	}
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
	paramsFromContext(ctx).ResultMetrics = metrics
	onResultMetricsChange := paramsFromContext(ctx).OnResultMetricsChange
	if onResultMetricsChange != nil {
		onResultMetricsChange(metrics)
	}
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
	paramsFromContext(ctx).ResultLogs = logs
	onResultLogsChange := paramsFromContext(ctx).OnResultLogsChange
	if onResultLogsChange != nil {
		onResultLogsChange(logs)
	}
}

func setStatusReasonFn(ctx context.Context, mod api.Module, stack []uint64) {
	// Read buffer pointer and size from the stack
	buf := uint32(stack[0])
	size := uint32(stack[1])

	// Read the status reason string from WASM memory
	reasonBytes, ok := mod.Memory().Read(buf, size)
	if !ok {
		panic("out of memory reading status reason") // Bug: caller passed a length outside memory
	}

	// Store the status reason in context
	paramsFromContext(ctx).StatusReason = string(reasonBytes)
}

// instantiateHostModule creates and instantiates the host module with exported functions
func instantiateHostModule(ctx context.Context, runtime wazero.Runtime) (api.Module, error) {
	return runtime.NewHostModuleBuilder(otelWasm).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultTracesFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(setResultTraces).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultTracesFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(legacySetResultTraces).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultMetricsFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(setResultMetrics).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultMetricsFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(legacySetResultMetrics).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultLogsFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(setResultLogs).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setResultLogsFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(legacySetResultLogs).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(getPluginConfigFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		WithParameterNames("buf", "buf_limit").Export(getPluginConfig).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(getPluginConfigFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		WithParameterNames("buf", "buf_limit").Export(legacyGetPluginConfig).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setStatusReasonFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(setStatusReason).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(setStatusReasonFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(legacySetResultStatusReason).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(getShutdownRequestedFn), []api.ValueType{}, []api.ValueType{api.ValueTypeI32}).
		Export(getShutdownRequested).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(getShutdownRequestedFn), []api.ValueType{}, []api.ValueType{api.ValueTypeI32}).
		Export(legacyGetShutdownRequested).
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
