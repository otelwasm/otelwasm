package main

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	"github.com/otelwasm/otelwasm/guest/plugin" // register processors
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

func init() {
	logger, err := zap.NewDevelopment()
	if err != nil {
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
}

func main() {}
