package wasmprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

var (
	typeStr                                = component.MustNewType("wasm")
	processorCapabilities                  = consumer.Capabilities{MutatesData: true}
	_                     component.Config = (*Config)(nil)
)

func createDefaultConfig() component.Config {
	cfg := &Config{}
	cfg.RuntimeConfig.Default()
	return cfg
}

// NewFactory creates a factory for wasmprocessor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithTraces(createTraces, component.StabilityLevelAlpha),
		processor.WithMetrics(createMetrics, component.StabilityLevelAlpha),
		processor.WithLogs(createLogs, component.StabilityLevelAlpha),
	)
}

func createTraces(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	wasmProcessor, err := newWasmTracesProcessor(ctx, cfg.(*Config), set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewTraces(ctx, set, cfg, nextConsumer,
		wasmProcessor.processTraces,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(wasmProcessor.shutdown),
	)
}

func createMetrics(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	wasmProcessor, err := newWasmMetricsProcessor(ctx, cfg.(*Config), set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewMetrics(ctx, set, cfg, nextConsumer,
		wasmProcessor.processMetrics,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(wasmProcessor.shutdown),
	)
}

func createLogs(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	wasmProcessor, err := newWasmLogsProcessor(ctx, cfg.(*Config), set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewLogs(ctx, set, cfg, nextConsumer,
		wasmProcessor.processLogs,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithShutdown(wasmProcessor.shutdown),
	)
}
