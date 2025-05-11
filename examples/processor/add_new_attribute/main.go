package main

import (
	attributeprocessor "github.com/otelwasm/otelwasm/examples/processor/add_new_attribute/processor"
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	guestplugin "github.com/otelwasm/otelwasm/guest/plugin" // register processors
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	// Create the factory from our implementation
	factory := attributeprocessor.NewFactory()
	telemetrySettings := componenttest.NewNopTelemetrySettings()
	telemetrySettings.Logger = logger

	settings := processor.Settings{
		ID:                component.MustNewID("add_new_attribute"),
		TelemetrySettings: telemetrySettings,
		BuildInfo:         component.NewDefaultBuildInfo(),
	}

	// Create a processor connector that wraps the factory
	connector := factoryconnector.NewProcessorConnector(factory, settings)

	// Register the processor for traces, metrics, and logs
	guestplugin.Set(struct {
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
