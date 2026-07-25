// Package wasmplugin provides common functionality for WebAssembly-based OpenTelemetry components.
package wasmplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync/atomic"

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
	// PluginConfigJSON is the JSON representation of the plugin config
	PluginConfigJSON []byte

	// runtime is the underlying WebAssembly runtime hosting the guest module
	runtime guestRuntime
}

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

	var runtime guestRuntime
	switch cfg.RuntimeConfig.Name {
	// An empty name falls back to the default runtime (wazero) for
	// backward compatibility.
	case "", RuntimeNameWazero:
		runtime, err = newWazeroRuntime(ctx, bytes, cfg.RuntimeConfig)
	case RuntimeNameWasmEdge:
		runtime, err = newWasmEdgeRuntime(ctx, bytes, cfg.RuntimeConfig)
	default:
		return nil, fmt.Errorf("wasm: invalid runtime name: %s", cfg.RuntimeConfig.Name)
	}
	if err != nil {
		return nil, err
	}

	// Check if all required and built-in guest functions are exported
	for _, funcName := range append(requiredFunctions, builtInGuestFunctions...) {
		if !runtime.HasFunction(funcName) {
			_ = runtime.Close(ctx)
			return nil, fmt.Errorf("wasm: %s is not exported: %w", funcName, ErrRequiredFunctionNotExported)
		}
	}

	// Convert the plugin config to JSON representation
	pluginConfigJSON, err := json.Marshal(cfg.PluginConfig)
	if err != nil {
		_ = runtime.Close(ctx)
		return nil, fmt.Errorf("wasm: error marshalling plugin config: %w", err)
	}

	plugin := &WasmPlugin{
		PluginConfigJSON: pluginConfigJSON,
		runtime:          runtime,
	}

	return plugin, nil
}

// ProcessFunctionCall executes a WASM function and handles stack management
func (p *WasmPlugin) ProcessFunctionCall(ctx context.Context, functionName string, stack *Stack) ([]uint64, error) {
	return p.runtime.CallFunction(ctx, functionName, stack)
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

// Shutdown closes the WASM runtime
func (p *WasmPlugin) Shutdown(ctx context.Context) error {
	return p.runtime.Close(ctx)
}
