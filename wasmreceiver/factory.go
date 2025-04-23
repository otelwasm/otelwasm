package wasmreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var (
	typeStr                               = component.MustNewType("wasm")
	receiverCapabilities                  = consumer.Capabilities{MutatesData: true}
	_                    component.Config = (*Config)(nil)
)

func createDefaultConfig() component.Config {
	return &Config{}
}

// NewFactory creates a factory for wasmreceiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetrics, component.StabilityLevelAlpha),
		receiver.WithLogs(createLogs, component.StabilityLevelAlpha),
		receiver.WithTraces(createTraces, component.StabilityLevelAlpha),
	)
}

func createMetrics(
	ctx context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	_, wasmreceiver, err := newMetricsWasmReceiver(ctx, cfg.(*Config), nextConsumer, set)
	if err != nil {
		return nil, err
	}
	return wasmreceiver, nil
}

func createLogs(
	ctx context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	_, wasmreceiver, err := newLogsWasmReceiver(ctx, cfg.(*Config), nextConsumer, set)
	if err != nil {
		return nil, err
	}
	return wasmreceiver, nil
}

func createTraces(
	ctx context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	_, wasmreceiver, err := newTracesWasmReceiver(ctx, cfg.(*Config), nextConsumer, set)
	if err != nil {
		return nil, err
	}
	return wasmreceiver, nil
}
