package wasmexporter

import (
	"context"
	"fmt"

	"github.com/otelwasm/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
)

const (
	pushTracesFunctionName  = "pushTraces"
	pushMetricsFunctionName = "pushMetrics"
	pushLogsFunctionName    = "pushLogs"
)

type wasmExporter struct {
	plugin *wasmplugin.WasmPlugin
}

// newWasmTracesExporter creates a new traces exporter using WebAssembly
func newWasmTracesExporter(ctx context.Context, cfg *Config) (*wasmExporter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Specify required functions for the traces exporter
	requiredFunctions := []string{pushTracesFunctionName}

	// Initialize the WASM plugin
	plugin, err := wasmplugin.NewWasmPlugin(ctx, &cfg.Config, requiredFunctions)
	if err != nil {
		return nil, err
	}

	// Check if traces are supported
	if supported, err := plugin.IsTracesSupported(ctx); err != nil {
		return nil, fmt.Errorf("failed to check traces support status: %w", err)
	} else if !supported {
		return nil, pipeline.ErrSignalNotSupported
	}

	return &wasmExporter{
		plugin: plugin,
	}, nil
}

// newWasmMetricsExporter creates a new metrics exporter using WebAssembly
func newWasmMetricsExporter(ctx context.Context, cfg *Config) (*wasmExporter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Specify required functions for the metrics exporter
	requiredFunctions := []string{pushMetricsFunctionName}

	// Initialize the WASM plugin
	plugin, err := wasmplugin.NewWasmPlugin(ctx, &cfg.Config, requiredFunctions)
	if err != nil {
		return nil, err
	}

	// Check if metrics are supported
	if supported, err := plugin.IsMetricsSupported(ctx); err != nil {
		return nil, fmt.Errorf("failed to check metrics support status: %w", err)
	} else if !supported {
		return nil, pipeline.ErrSignalNotSupported
	}

	return &wasmExporter{
		plugin: plugin,
	}, nil
}

// newWasmLogsExporter creates a new logs exporter using WebAssembly
func newWasmLogsExporter(ctx context.Context, cfg *Config) (*wasmExporter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Specify required functions for the logs exporter
	requiredFunctions := []string{pushLogsFunctionName}

	// Initialize the WASM plugin
	plugin, err := wasmplugin.NewWasmPlugin(ctx, &cfg.Config, requiredFunctions)
	if err != nil {
		return nil, err
	}

	// Check if logs are supported
	if supported, err := plugin.IsLogsSupported(ctx); err != nil {
		return nil, fmt.Errorf("failed to check logs support status: %w", err)
	} else if !supported {
		return nil, pipeline.ErrSignalNotSupported
	}

	return &wasmExporter{
		plugin: plugin,
	}, nil
}

func (wp *wasmExporter) pushTraces(
	ctx context.Context,
	td ptrace.Traces,
) error {
	stack := &wasmplugin.Stack{
		CurrentTraces:    td,
		PluginConfigJSON: wp.plugin.PluginConfigJSON,
	}

	res, err := wp.plugin.ProcessFunctionCall(ctx, pushTracesFunctionName, stack)
	if err != nil {
		return err
	}

	statusCode := wasmplugin.StatusCode(res[0])
	if statusCode != 0 {
		return fmt.Errorf("wasm: error pushing traces: %s: %s", statusCode.String(), stack.StatusReason)
	}

	return nil
}

func (wp *wasmExporter) pushMetrics(
	ctx context.Context,
	md pmetric.Metrics,
) error {
	stack := &wasmplugin.Stack{
		CurrentMetrics:   md,
		PluginConfigJSON: wp.plugin.PluginConfigJSON,
	}

	res, err := wp.plugin.ProcessFunctionCall(ctx, pushMetricsFunctionName, stack)
	if err != nil {
		return err
	}

	statusCode := wasmplugin.StatusCode(res[0])
	if statusCode != 0 {
		return fmt.Errorf("wasm: error pushing metrics: %s: %s", statusCode.String(), stack.StatusReason)
	}

	return nil
}

func (wp *wasmExporter) pushLogs(
	ctx context.Context,
	ld plog.Logs,
) error {
	stack := &wasmplugin.Stack{
		CurrentLogs:      ld,
		PluginConfigJSON: wp.plugin.PluginConfigJSON,
	}

	res, err := wp.plugin.ProcessFunctionCall(ctx, pushLogsFunctionName, stack)
	if err != nil {
		return err
	}

	statusCode := wasmplugin.StatusCode(res[0])
	if statusCode != 0 {
		return fmt.Errorf("wasm: error pushing logs: %s: %s", statusCode.String(), stack.StatusReason)
	}

	return nil
}

func (wp *wasmExporter) shutdown(ctx context.Context) error {
	return wp.plugin.Shutdown(ctx)
}
