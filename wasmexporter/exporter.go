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
	consumeTracesFunctionName  = "consume_traces"
	consumeMetricsFunctionName = "consume_metrics"
	consumeLogsFunctionName    = "consume_logs"
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
	requiredFunctions := []string{consumeTracesFunctionName}

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
	requiredFunctions := []string{consumeMetricsFunctionName}

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
	requiredFunctions := []string{consumeLogsFunctionName}

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
	_, err := wp.plugin.ConsumeTraces(ctx, td)
	return err
}

func (wp *wasmExporter) pushMetrics(
	ctx context.Context,
	md pmetric.Metrics,
) error {
	_, err := wp.plugin.ConsumeMetrics(ctx, md)
	return err
}

func (wp *wasmExporter) pushLogs(
	ctx context.Context,
	ld plog.Logs,
) error {
	_, err := wp.plugin.ConsumeLogs(ctx, ld)
	return err
}

func (wp *wasmExporter) shutdown(ctx context.Context) error {
	return wp.plugin.Shutdown(ctx)
}
