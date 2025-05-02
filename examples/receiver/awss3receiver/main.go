package main

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awss3receiver"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	"github.com/otelwasm/otelwasm/guest/plugin" // register receivers
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	factory := awss3receiver.NewFactory()
	telemetrySettings := componenttest.NewNopTelemetrySettings()
	telemetrySettings.Logger = logger

	settings := receiver.Settings{
		ID:                component.MustNewID("awss3"),
		TelemetrySettings: telemetrySettings,
		BuildInfo:         component.NewDefaultBuildInfo(),
	}

	connector := factoryconnector.NewReceiverConnector(factory, settings)

	plugin.Set(connector.Metrics())
	plugin.Set(connector.Logs())
	plugin.Set(connector.Traces())
}
func main() {}
