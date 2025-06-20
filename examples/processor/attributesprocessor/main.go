package main

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	"github.com/otelwasm/otelwasm/guest/logging"
	"github.com/otelwasm/otelwasm/guest/plugin" // register processors
	"github.com/otelwasm/otelwasm/guest/telemetry"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

func init() {
	// Log processor initialization
	logging.Info("Initializing attributes processor plugin")

	logger, err := zap.NewDevelopment()
	if err != nil {
		logging.Error("Failed to create logger", map[string]string{
			"error": err.Error(),
		})
		panic(err)
	}

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

	// Get telemetry settings from host to enrich logging
	serviceName := telemetry.GetServiceName()
	serviceVersion := telemetry.GetServiceVersion()
	
	logging.Info("Attributes processor plugin initialized successfully", map[string]string{
		"processor_id":    "attributes",
		"supports":        "traces,metrics,logs",
		"service_name":    serviceName,
		"service_version": serviceVersion,
	})

	// Log resource attributes for debugging
	attrs := telemetry.GetAllResourceAttributes()
	if len(attrs) > 0 {
		logging.Debug("Resource attributes available", map[string]string{
			"attribute_count": string(rune(len(attrs))),
		})
	}
}

func main() {}
