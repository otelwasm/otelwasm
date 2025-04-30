package wasmprocessor

import (
	"context"
	"fmt"

	"github.com/musaprg/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
)

const (
	processTracesFunctionName  = "processTraces"
	processMetricsFunctionName = "processMetrics"
	processLogsFunctionName    = "processLogs"
)

type wasmProcessor struct {
	plugin *wasmplugin.WasmPlugin
}

func newWasmMetricsProcessor(ctx context.Context, cfg *Config) (*wasmProcessor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Specify required functions for the processor
	requiredFunctions := []string{processMetricsFunctionName}

	// Initialize the WASM plugin
	plugin, err := wasmplugin.NewWasmPlugin(ctx, &cfg.Config, requiredFunctions)
	if err != nil {
		return nil, err
	}

	if supported, err := plugin.IsMetricsSupported(ctx); err != nil {
		return nil, fmt.Errorf("failed to check metrics support status: %w", err)
	} else if !supported {
		return nil, pipeline.ErrSignalNotSupported
	}

	return &wasmProcessor{
		plugin: plugin,
	}, nil
}

func newWasmLogsProcessor(ctx context.Context, cfg *Config) (*wasmProcessor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Specify required functions for the processor
	requiredFunctions := []string{processLogsFunctionName}

	// Initialize the WASM plugin
	plugin, err := wasmplugin.NewWasmPlugin(ctx, &cfg.Config, requiredFunctions)
	if err != nil {
		return nil, err
	}

	if supported, err := plugin.IsLogsSupported(ctx); err != nil {
		return nil, fmt.Errorf("failed to check logs support status: %w", err)
	} else if !supported {
		return nil, pipeline.ErrSignalNotSupported
	}

	return &wasmProcessor{
		plugin: plugin,
	}, nil
}

func newWasmTracesProcessor(ctx context.Context, cfg *Config) (*wasmProcessor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Specify required functions for the processor
	requiredFunctions := []string{processTracesFunctionName}

	// Initialize the WASM plugin
	plugin, err := wasmplugin.NewWasmPlugin(ctx, &cfg.Config, requiredFunctions)
	if err != nil {
		return nil, err
	}

	if supported, err := plugin.IsTracesSupported(ctx); err != nil {
		return nil, fmt.Errorf("failed to check traces support status: %w", err)
	} else if !supported {
		return nil, pipeline.ErrSignalNotSupported
	}

	return &wasmProcessor{
		plugin: plugin,
	}, nil
}

func (wp *wasmProcessor) processTraces(
	ctx context.Context,
	td ptrace.Traces,
) (ptrace.Traces, error) {
	stack := &wasmplugin.Stack{
		CurrentTraces:    td,
		PluginConfigJSON: wp.plugin.PluginConfigJSON,
	}

	res, err := wp.plugin.ProcessFunctionCall(ctx, processTracesFunctionName, stack)
	if err != nil {
		return td, err
	}

	statusCode := wasmplugin.StatusCode(res[0])
	if statusCode != 0 {
		return td, fmt.Errorf("wasm: error processing traces: %s: %s", statusCode.String(), stack.StatusReason)
	}

	return stack.ResultTraces, nil
}

func (wp *wasmProcessor) processMetrics(
	ctx context.Context,
	md pmetric.Metrics,
) (pmetric.Metrics, error) {
	stack := &wasmplugin.Stack{
		CurrentMetrics:   md,
		PluginConfigJSON: wp.plugin.PluginConfigJSON,
	}

	res, err := wp.plugin.ProcessFunctionCall(ctx, processMetricsFunctionName, stack)
	if err != nil {
		return md, err
	}

	statusCode := wasmplugin.StatusCode(res[0])
	if statusCode != 0 {
		return md, fmt.Errorf("wasm: error processing metrics: %s: %s", statusCode.String(), stack.StatusReason)
	}

	return stack.ResultMetrics, nil
}

func (wp *wasmProcessor) processLogs(
	ctx context.Context,
	ld plog.Logs,
) (plog.Logs, error) {
	stack := &wasmplugin.Stack{
		CurrentLogs:      ld,
		PluginConfigJSON: wp.plugin.PluginConfigJSON,
	}

	res, err := wp.plugin.ProcessFunctionCall(ctx, processLogsFunctionName, stack)
	if err != nil {
		return ld, err
	}

	statusCode := wasmplugin.StatusCode(res[0])
	if statusCode != 0 {
		return ld, fmt.Errorf("wasm: error processing logs: %s: %s", statusCode.String(), stack.StatusReason)
	}

	return stack.ResultLogs, nil
}

func (wp *wasmProcessor) shutdown(ctx context.Context) error {
	return wp.plugin.Shutdown(ctx)
}
