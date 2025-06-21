package main

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/webhookeventreceiver"
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	"github.com/otelwasm/otelwasm/guest/logging"
	"github.com/otelwasm/otelwasm/guest/plugin" // register receivers
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
)

func init() {
	// Log receiver initialization
	logging.Info("Initializing webhook event receiver plugin")

	// Use host bridge logger instead of creating a new zap logger
	// This ensures all logging goes through the host-side logger
	logger := logging.NewHostBridgeLogger()

	factory := webhookeventreceiver.NewFactory()
	telemetrySettings := componenttest.NewNopTelemetrySettings()
	telemetrySettings.Logger = logger

	settings := receiver.Settings{
		ID:                component.MustNewID("webhookevent"),
		TelemetrySettings: telemetrySettings,
		BuildInfo:         component.NewDefaultBuildInfo(),
	}

	connector := factoryconnector.NewReceiverConnector(factory, settings)

	plugin.Set(struct {
		api.LogsReceiver
	}{
		connector.Logs(),
	})

	logging.Info("Webhook event receiver plugin initialized successfully", map[string]string{
		"receiver_id": "webhookevent",
		"supports":    "logs",
	})
}
func main() {}
