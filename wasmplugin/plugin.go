// Package wasmplugin provides common functionality for WebAssembly-based OpenTelemetry components.
package wasmplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/otelwasm/otelwasm/runtime"
	_ "github.com/otelwasm/otelwasm/runtime/wazero" // Register Wazero runtime
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
	currentTraces         = "currentTraces"
	currentMetrics        = "currentMetrics"
	currentLogs           = "currentLogs"
	setResultTraces       = "setResultTraces"
	setResultMetrics      = "setResultMetrics"
	setResultLogs         = "setResultLogs"
	getPluginConfig       = "getPluginConfig"
	setResultStatusReason = "setResultStatusReason"
	getShutdownRequested  = "getShutdownRequested"

	// Guest function
	getSupportedTelemetry = "getSupportedTelemetry"

	// WASI extension name
	wasmEdgeV2Extension = "wasmedgev2"
)

var builtInGuestFunctions = []string{
	getSupportedTelemetry,
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
	// Runtime is the WebAssembly runtime (abstracted)
	Runtime runtime.Runtime

	// RuntimeContext holds runtime-specific state (WASI, host modules, etc.)
	RuntimeContext runtime.Context

	// Module is the instantiated WASM module (abstracted)
	Module runtime.ModuleInstance

	// PluginConfigJSON is the JSON representation of the plugin config
	PluginConfigJSON []byte

	// Exported functions from the WASM module (abstracted)
	ExportedFunctions map[string]runtime.FunctionInstance
}

// stackKey is the key used to store the stack in the context
type stackKey struct{}

// Stack holds the data being passed between the host and the guest
type Stack struct {
	CurrentTraces     ptrace.Traces
	CurrentMetrics    pmetric.Metrics
	CurrentLogs       plog.Logs
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

	// Create runtime using the new abstraction
	rt, err := runtime.NewRuntime(cfg.RuntimeConfig.Type, cfg.RuntimeConfig)
	if err != nil {
		return nil, fmt.Errorf("wasm: error creating runtime: %w", err)
	}

	// Compile the WASM module
	compiledModule, err := rt.Compile(ctx, bytes)
	if err != nil {
		return nil, fmt.Errorf("wasm: error compiling module: %w", err)
	}

	// Create host module with OTel host functions
	hostModule := createOTelHostModule()

	// Instantiate module with host functions
	moduleInstance, runtimeContext, err := rt.InstantiateWithHost(ctx, compiledModule, *hostModule)
	if err != nil {
		return nil, fmt.Errorf("wasm: error instantiating module: %w", err)
	}

	// Get exported functions
	exportedFunctions := make(map[string]runtime.FunctionInstance)
	for _, funcName := range requiredFunctions {
		fn := moduleInstance.Function(funcName)
		if fn == nil {
			return nil, fmt.Errorf("wasm: %s is not exported: %w", funcName, ErrRequiredFunctionNotExported)
		}
		exportedFunctions[funcName] = fn
	}

	// Check built-in guest functions
	for _, funcName := range builtInGuestFunctions {
		fn := moduleInstance.Function(funcName)
		if fn == nil {
			return nil, fmt.Errorf("wasm: %s is not exported: %w", funcName, ErrRequiredFunctionNotExported)
		}
		exportedFunctions[funcName] = fn
	}

	// Convert the plugin config to JSON representation
	pluginConfigJSON, err := json.Marshal(cfg.PluginConfig)
	if err != nil {
		return nil, fmt.Errorf("wasm: error marshalling plugin config: %w", err)
	}

	plugin := &WasmPlugin{
		Runtime:           rt,
		RuntimeContext:    runtimeContext,
		Module:            moduleInstance,
		PluginConfigJSON:  pluginConfigJSON,
		ExportedFunctions: exportedFunctions,
	}

	return plugin, nil
}

// createOTelHostModule creates a host module with all OpenTelemetry functions
func createOTelHostModule() *runtime.HostModule {
	hostModule := &runtime.HostModule{
		Name: otelWasm,
		Functions: []runtime.HostFunctionDefinition{
			{
				FunctionName: currentTraces,
				Function:     &runtime.WazeroHostFunction{Function: currentTracesFn},
				ParamTypes:   []runtime.ValueType{runtime.ValueTypeI32, runtime.ValueTypeI32},
				ResultTypes:  []runtime.ValueType{runtime.ValueTypeI32},
			},
			{
				FunctionName: currentMetrics,
				Function:     &runtime.WazeroHostFunction{Function: currentMetricsFn},
				ParamTypes:   []runtime.ValueType{runtime.ValueTypeI32, runtime.ValueTypeI32},
				ResultTypes:  []runtime.ValueType{runtime.ValueTypeI32},
			},
			{
				FunctionName: currentLogs,
				Function:     &runtime.WazeroHostFunction{Function: currentLogsFn},
				ParamTypes:   []runtime.ValueType{runtime.ValueTypeI32, runtime.ValueTypeI32},
				ResultTypes:  []runtime.ValueType{runtime.ValueTypeI32},
			},
			{
				FunctionName: setResultTraces,
				Function:     &runtime.WazeroHostFunction{Function: setResultTracesFn},
				ParamTypes:   []runtime.ValueType{runtime.ValueTypeI32, runtime.ValueTypeI32},
				ResultTypes:  []runtime.ValueType{},
			},
			{
				FunctionName: setResultMetrics,
				Function:     &runtime.WazeroHostFunction{Function: setResultMetricsFn},
				ParamTypes:   []runtime.ValueType{runtime.ValueTypeI32, runtime.ValueTypeI32},
				ResultTypes:  []runtime.ValueType{},
			},
			{
				FunctionName: setResultLogs,
				Function:     &runtime.WazeroHostFunction{Function: setResultLogsFn},
				ParamTypes:   []runtime.ValueType{runtime.ValueTypeI32, runtime.ValueTypeI32},
				ResultTypes:  []runtime.ValueType{},
			},
			{
				FunctionName: getPluginConfig,
				Function:     &runtime.WazeroHostFunction{Function: getPluginConfigFn},
				ParamTypes:   []runtime.ValueType{runtime.ValueTypeI32, runtime.ValueTypeI32},
				ResultTypes:  []runtime.ValueType{runtime.ValueTypeI32},
			},
			{
				FunctionName: setResultStatusReason,
				Function:     &runtime.WazeroHostFunction{Function: setResultStatusReasonFn},
				ParamTypes:   []runtime.ValueType{runtime.ValueTypeI32, runtime.ValueTypeI32},
				ResultTypes:  []runtime.ValueType{},
			},
			{
				FunctionName: getShutdownRequested,
				Function:     &runtime.WazeroHostFunction{Function: getShutdownRequestedFn},
				ParamTypes:   []runtime.ValueType{},
				ResultTypes:  []runtime.ValueType{runtime.ValueTypeI32},
			},
		},
	}
	return hostModule
}

// createContextWithStack creates a new context with a Stack
func createContextWithStack(ctx context.Context, stack *Stack) context.Context {
	return context.WithValue(ctx, stackKey{}, stack)
}

// ProcessFunctionCall executes a WASM function and handles stack management
func (p *WasmPlugin) ProcessFunctionCall(ctx context.Context, functionName string, stack *Stack) ([]uint64, error) {
	ctx = createContextWithStack(ctx, stack)
	// Set runtime context with WASI module instance for function calls
	ctx = p.RuntimeContext.WithRuntimeContext(ctx)

	fn, ok := p.ExportedFunctions[functionName]
	if !ok {
		return nil, fmt.Errorf("wasm: function not found: %s", functionName)
	}

	return fn.Call(ctx)
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
	// Close runtime context first
	if p.RuntimeContext != nil {
		if err := p.RuntimeContext.Close(ctx); err != nil {
			return fmt.Errorf("wasm: error closing runtime context: %w", err)
		}
	}

	// Close the runtime
	if p.Runtime != nil {
		if err := p.Runtime.Close(ctx); err != nil {
			return fmt.Errorf("wasm: error closing runtime: %w", err)
		}
	}

	return nil
}

// Host function implementations
func currentTracesFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := uint32(stack[1])

	traces := paramsFromContext(ctx).CurrentTraces
	stack[0] = uint64(marshalTraceIfUnderLimit(mod.Memory(), traces, buf, bufLimit))
}

func currentMetricsFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := uint32(stack[1])

	metrics := paramsFromContext(ctx).CurrentMetrics
	stack[0] = uint64(marshalMetricsIfUnderLimit(mod.Memory(), metrics, buf, bufLimit))
}

func currentLogsFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := uint32(stack[1])

	logs := paramsFromContext(ctx).CurrentLogs
	stack[0] = uint64(marshalLogsIfUnderLimit(mod.Memory(), logs, buf, bufLimit))
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
	paramsFromContext(ctx).StatusReason = string(reasonBytes)
}

// Legacy functions removed - these were Wazero-specific WASI context management
// TODO: These will be handled by the runtime abstraction layer in the future
