package wasmexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

var (
	typeStr                               = component.MustNewType("wasm")
	exporterCapabilities                  = consumer.Capabilities{MutatesData: true}
	_                    component.Config = (*Config)(nil)
)

func createDefaultConfig() component.Config {
	cfg := &Config{}
	cfg.RuntimeConfig.Default()
	return cfg
}

// NewFactory creates a factory for wasmexporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithTraces(createTraces, component.StabilityLevelAlpha),
		exporter.WithMetrics(createMetrics, component.StabilityLevelAlpha),
		exporter.WithLogs(createLogs, component.StabilityLevelAlpha),
	)
}

func createTraces(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	wasmExporter, err := newWasmTracesExporter(ctx, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewTraces(ctx, set, cfg,
		wasmExporter.pushTraces,
		exporterhelper.WithCapabilities(exporterCapabilities),
		exporterhelper.WithShutdown(wasmExporter.shutdown),
	)
}

func createMetrics(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	wasmExporter, err := newWasmMetricsExporter(ctx, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewMetrics(ctx, set, cfg,
		wasmExporter.pushMetrics,
		exporterhelper.WithCapabilities(exporterCapabilities),
		exporterhelper.WithShutdown(wasmExporter.shutdown),
	)
}

func createLogs(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	wasmExporter, err := newWasmLogsExporter(ctx, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewLogs(ctx, set, cfg,
		wasmExporter.pushLogs,
		exporterhelper.WithCapabilities(exporterCapabilities),
		exporterhelper.WithShutdown(wasmExporter.shutdown),
	)
}
