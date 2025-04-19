package wasmexporter

import (
	"context"
	"fmt"

	"github.com/musaprg/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type wasmExporter struct {
	plugin *wasmplugin.WasmPlugin
}

func newWasmExporter(ctx context.Context, cfg *Config) (*wasmExporter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Specify required functions for the exporter
	requiredFunctions := []string{"pushTraces", "pushMetrics", "pushLogs"}

	// Create a wasmplugin configuration from our exporter config
	pluginCfg := &wasmplugin.Config{
		Path:         cfg.Path,
		PluginConfig: cfg.PluginConfig,
	}

	// Initialize the WASM plugin
	plugin, err := wasmplugin.NewWasmPlugin(ctx, pluginCfg, requiredFunctions)
	if err != nil {
		return nil, err
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

	res, err := wp.plugin.ProcessFunctionCall(ctx, "pushTraces", stack)
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

	res, err := wp.plugin.ProcessFunctionCall(ctx, "pushMetrics", stack)
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

	res, err := wp.plugin.ProcessFunctionCall(ctx, "pushLogs", stack)
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
