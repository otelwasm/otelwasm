// Package wasmplugin provides common functionality for WebAssembly-based OpenTelemetry components.
package wasmplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/stealthrocket/wasi-go"
	wasigo "github.com/stealthrocket/wasi-go/imports"
	"github.com/stealthrocket/wasi-go/imports/wasi_snapshot_preview1"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	logMessage            = "logMessage"
	getTelemetrySettings  = "getTelemetrySettings"

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
	// Runtime is the WebAssembly runtime
	Runtime wazero.Runtime

	// System is the WASI system implementation
	Sys wasi.System

	// Module is the instantiated WASM module
	Module api.Module

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

	// Logger is the host-side logger for the WASM plugin
	Logger *zap.Logger

	// TelemetrySettings contains the complete telemetry settings for the component
	TelemetrySettings component.TelemetrySettings
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
	for _, funcName := range builtInGuestFunctions {
		fn := mod.ExportedFunction(funcName)
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
		Runtime:           runtime,
		Sys:               sys,
		Module:            mod,
		PluginConfigJSON:  pluginConfigJSON,
		ExportedFunctions: exportedFunctions,
		wasiP1HostModule:  wasiP1HostModule,
	}

	return plugin, nil
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
	ctx = createContextWithStack(ctx, stack)
	// Set the WASI host module instance in the context
	ctx = withModuleInstance(ctx, p.wasiP1HostModule)

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
	if err := p.Sys.Close(ctx); err != nil {
		return fmt.Errorf("wasm: error closing system: %w", err)
	}
	if err := p.Runtime.Close(ctx); err != nil {
		return fmt.Errorf("wasm: error closing runtime: %w", err)
	}
	return nil
}

// LogMessage represents a structured log message from the guest
type LogMessage struct {
	Level   int32             `json:"level"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

// SerializableTelemetrySettings represents telemetry settings that can be serialized to WASM
type SerializableTelemetrySettings struct {
	// Resource attributes as a map
	ResourceAttributes map[string]interface{} `json:"resource_attributes"`
	// Service name extracted from resource
	ServiceName string `json:"service_name"`
	// Service version extracted from resource  
	ServiceVersion string `json:"service_version"`
	// Component ID information
	ComponentID map[string]string `json:"component_id"`
}

// zapLevelFromSlogLevel converts slog.Level to zapcore.Level
func zapLevelFromSlogLevel(level slog.Level) zapcore.Level {
	switch {
	case level <= slog.LevelDebug:
		return zapcore.DebugLevel
	case level <= slog.LevelInfo:
		return zapcore.InfoLevel
	case level <= slog.LevelWarn:
		return zapcore.WarnLevel
	default:
		return zapcore.ErrorLevel
	}
}

// Host function implementations
func logMessageFn(ctx context.Context, mod api.Module, stack []uint64) {
	// Read buffer pointer and size from the stack
	buf := uint32(stack[0])
	size := uint32(stack[1])

	// Read the serialized log message from WASM memory
	logBytes, ok := mod.Memory().Read(buf, size)
	if !ok {
		panic("out of memory reading log message") // Bug: caller passed a length outside memory
	}

	// Unmarshal the log message
	var logMsg LogMessage
	if err := json.Unmarshal(logBytes, &logMsg); err != nil {
		// If we can't unmarshal, log the raw message as a fallback
		if logger := paramsFromContext(ctx).Logger; logger != nil {
			logger.Error("failed to unmarshal log message from guest", zap.String("raw_message", string(logBytes)), zap.Error(err))
		}
		return
	}

	// Get the logger from context
	logger := paramsFromContext(ctx).Logger
	if logger == nil {
		// No logger available, skip logging
		return
	}

	// Convert slog level to zap level
	zapLevel := zapLevelFromSlogLevel(slog.Level(logMsg.Level))

	// Create zap fields from the structured fields
	fields := make([]zap.Field, 0, len(logMsg.Fields))
	for key, value := range logMsg.Fields {
		fields = append(fields, zap.String(key, value))
	}

	// Log the message at the appropriate level
	switch zapLevel {
	case zapcore.DebugLevel:
		logger.Debug(logMsg.Message, fields...)
	case zapcore.InfoLevel:
		logger.Info(logMsg.Message, fields...)
	case zapcore.WarnLevel:
		logger.Warn(logMsg.Message, fields...)
	case zapcore.ErrorLevel:
		logger.Error(logMsg.Message, fields...)
	default:
		logger.Info(logMsg.Message, fields...)
	}
}

// telemetrySettingsToSerializable converts component.TelemetrySettings to SerializableTelemetrySettings
func telemetrySettingsToSerializable(ts component.TelemetrySettings) SerializableTelemetrySettings {
	serializable := SerializableTelemetrySettings{
		ResourceAttributes: make(map[string]interface{}),
		ComponentID:        make(map[string]string),
	}

	// Extract resource attributes
	if ts.Resource.Len() > 0 {
		ts.Resource.Attributes().Range(func(k string, v pcommon.Value) bool {
			switch v.Type() {
			case pcommon.ValueTypeStr:
				serializable.ResourceAttributes[k] = v.Str()
			case pcommon.ValueTypeInt:
				serializable.ResourceAttributes[k] = v.Int()
			case pcommon.ValueTypeBool:
				serializable.ResourceAttributes[k] = v.Bool()
			case pcommon.ValueTypeDouble:
				serializable.ResourceAttributes[k] = v.Double()
			default:
				serializable.ResourceAttributes[k] = v.AsString()
			}

			// Extract specific service attributes
			switch k {
			case "service.name":
				if v.Type() == pcommon.ValueTypeStr {
					serializable.ServiceName = v.Str()
				}
			case "service.version":
				if v.Type() == pcommon.ValueTypeStr {
					serializable.ServiceVersion = v.Str()
				}
			}
			return true
		})
	}

	return serializable
}

func getTelemetrySettingsFn(ctx context.Context, mod api.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := uint32(stack[1])

	telemetrySettings := paramsFromContext(ctx).TelemetrySettings
	serializable := telemetrySettingsToSerializable(telemetrySettings)

	// Marshal to JSON
	jsonBytes, err := json.Marshal(serializable)
	if err != nil {
		// If marshaling fails, write empty object
		jsonBytes = []byte("{}")
	}

	stack[0] = uint64(writeBytesIfUnderLimit(mod.Memory(), jsonBytes, buf, bufLimit))
}

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
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(logMessageFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		WithParameterNames("buf", "buf_len").Export(logMessage).
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(getTelemetrySettingsFn), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		WithParameterNames("buf", "buf_limit").Export(getTelemetrySettings).
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
