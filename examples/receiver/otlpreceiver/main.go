package main

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	"github.com/otelwasm/otelwasm/guest/plugin" // register receivers
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.uber.org/zap"
)

// TODO: Fix the bug when using the gRPC endpoint.
// Currently, the gRPC endpoint is not working properly due to the panic while handling the incoming request.
// For more details, see https://github.com/otelwasm/otelwasm/issues/59

func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	factory := otlpreceiver.NewFactory()
	telemetrySettings := componenttest.NewNopTelemetrySettings()
	telemetrySettings.Logger = logger

	settings := receiver.Settings{
		ID:                component.MustNewID("otlp"),
		TelemetrySettings: telemetrySettings,
		BuildInfo:         component.NewDefaultBuildInfo(),
	}

	connector := factoryconnector.NewReceiverConnector(factory, settings)

	plugin.Set(struct {
		api.MetricsReceiver
		api.LogsReceiver
		api.TracesReceiver
	}{
		connector.Metrics(),
		connector.Logs(),
		connector.Traces(),
	})
}
func main() {}
