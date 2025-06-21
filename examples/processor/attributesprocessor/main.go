package main

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	"github.com/otelwasm/otelwasm/guest/logging"
	"github.com/otelwasm/otelwasm/guest/plugin" // register processors
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/processor"
)

func init() {
	// Log processor initialization
	logging.Info("Initializing attributes processor plugin")

	// Use host bridge logger instead of creating a new zap logger
	// This ensures all logging goes through the host-side logger
	logger := logging.NewHostBridgeLogger()

	// Create the factory from the upstream implementation
	factory := attributesprocessor.NewFactory()
	telemetrySettings := componenttest.NewNopTelemetrySettings()
	telemetrySettings.Logger = logger

	settings := processor.Settings{
		ID:                component.MustNewID("attributes"),
		TelemetrySettings: telemetrySettings,
		BuildInfo:         component.NewDefaultBuildInfo(),
	}

	// Create a processor connector that wraps the factory
	connector := factoryconnector.NewProcessorConnector(factory, settings)

	// Register the processor for traces, metrics, and logs
	plugin.Set(struct {
		api.MetricsProcessor
		api.LogsProcessor
		api.TracesProcessor
	}{
		connector.Metrics(),
		connector.Logs(),
		connector.Traces(),
	})

	logging.Info("Attributes processor plugin initialized successfully", map[string]string{
		"processor_id": "attributes",
		"supports":     "traces,metrics,logs",
	})
}

func main() {}
