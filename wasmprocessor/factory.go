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
	return &Config{}
}

// NewFactory creates a factory for wasmprocessor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		typeStr,
		createDefaultConfig,
		processor.WithTraces(createTraces, component.StabilityLevelAlpha),
		// TODO: Implement Metrics and Logs processors
	)
}

func createTraces(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	wasmProcessor := &wasmProcessor{}
	return processorhelper.NewTraces(ctx, set, cfg, nextConsumer,
		wasmProcessor.processTraces,
		processorhelper.WithCapabilities(processorCapabilities),
	)
}
