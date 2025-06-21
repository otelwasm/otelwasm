package main

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awss3exporter"
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	"github.com/otelwasm/otelwasm/guest/logging"
	"github.com/otelwasm/otelwasm/guest/plugin" // register exporters
	"github.com/otelwasm/otelwasm/guest/telemetry"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
)

func init() {
	logging.Info("Initializing AWS S3 exporter plugin")

	// Use host bridge logger instead of creating a new zap logger
	logger := logging.NewHostBridgeLogger()

	factory := awss3exporter.NewFactory()
	telemetrySettings := componenttest.NewNopTelemetrySettings()
	telemetrySettings.Logger = logger

	settings := exporter.Settings{
		ID:                component.MustNewID("awss3"),
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

	// Log telemetry settings
	serviceName := telemetry.GetServiceName()
	logging.Info("AWS S3 exporter plugin initialized successfully", map[string]string{
		"exporter_id":    "awss3",
		"supports":       "traces,metrics,logs", 
		"service_name":   serviceName,
	})
}
func main() {}
