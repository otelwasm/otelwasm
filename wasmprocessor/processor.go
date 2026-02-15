package wasmprocessor

import (
	"context"
	"errors"
	"fmt"

	"github.com/otelwasm/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
)

const (
	consumeTracesFunctionName  = "consume_traces"
	processMetricsFunctionName = "processMetrics"
	processLogsFunctionName    = "processLogs"
	startFunctionName          = "start"
	shutdownFunctionName       = "shutdown"
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
	requiredFunctions := []string{consumeTracesFunctionName, startFunctionName, shutdownFunctionName}

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
	return wp.plugin.ConsumeTraces(ctx, td)
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
	var lifecycleErr error

	if _, ok := wp.plugin.ExportedFunctions[shutdownFunctionName]; ok {
		stack := &wasmplugin.Stack{PluginConfigJSON: wp.plugin.PluginConfigJSON}
		res, err := wp.plugin.ProcessFunctionCall(ctx, shutdownFunctionName, stack)
		if err != nil {
			lifecycleErr = err
		} else if len(res) == 0 {
			lifecycleErr = fmt.Errorf("wasm: %s returned no status code", shutdownFunctionName)
		} else {
			statusCode := wasmplugin.StatusCode(res[0])
			if statusCode != 0 {
				lifecycleErr = fmt.Errorf("wasm: error shutting down processor: %s: %s", statusCode.String(), stack.StatusReason)
			}
		}
	}

	runtimeErr := wp.plugin.Shutdown(ctx)
	return errors.Join(lifecycleErr, runtimeErr)
}

func (wp *wasmProcessor) start(ctx context.Context, _ component.Host) error {
	stack := &wasmplugin.Stack{PluginConfigJSON: wp.plugin.PluginConfigJSON}
	res, err := wp.plugin.ProcessFunctionCall(ctx, startFunctionName, stack)
	if err != nil {
		return err
	}
	if len(res) == 0 {
		return fmt.Errorf("wasm: %s returned no status code", startFunctionName)
	}
	statusCode := wasmplugin.StatusCode(res[0])
	if statusCode != 0 {
		return fmt.Errorf("wasm: error starting processor: %s: %s", statusCode.String(), stack.StatusReason)
	}
	return nil
}
