package main

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	"github.com/otelwasm/otelwasm/guest/plugin" // register exporters
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.uber.org/zap"
)

func init() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	factory := otlphttpexporter.NewFactory()
	telemetrySettings := componenttest.NewNopTelemetrySettings()
	telemetrySettings.Logger = logger

	settings := exporter.Settings{
		ID:                component.MustNewID("otlphttp"),
		TelemetrySettings: telemetrySettings,
		BuildInfo:         component.NewDefaultBuildInfo(),
	}

	connector := factoryconnector.NewExporterConnector(factory, settings)

	plugin.Set(struct {
		api.MetricsExporter
		api.LogsExporter
		api.TracesExporter
	}{
		connector.Metrics(),
		connector.Logs(),
		connector.Traces(),
	})
}

func main() {}
