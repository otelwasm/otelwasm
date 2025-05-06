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

// TODO: Fix the bug when we use sending queue on exporter.
// Currently, the exporter is not working properly with sending queue because of the architecture of the wasm plugin.
//
// Here's the example of the configuration:
//
// ```yaml
// exporters:
//   wasm/otlphttpexporter:
//     path: "/path/to/main.wasm"
//     plugin_config:
//       sending_queue:
//         enabled: false
// ```
//
// You MUST disable the sending queue when you use the otlphttpexporter built as a wasm module.
// For more details, see https://github.com/otelwasm/otelwasm/issues/60

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
