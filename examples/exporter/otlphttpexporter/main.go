package main

import (
	"context"

	"github.com/otelwasm/otelwasm/examples/exporter/otlphttpexporter/otlphttpexporter"
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	"github.com/otelwasm/otelwasm/guest/plugin" // register exporters
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

var _ api.TracesExporter = (*exp)(nil)

type exp struct {
	e exporter.Traces
}

func (e exp) PushTraces(td ptrace.Traces) *api.Status {
	if e.e == nil {
		return &api.Status{
			Code:   api.StatusCodeError,
			Reason: "exporter is not initialized",
		}
	}

	ctx := context.Background()
	err := e.e.ConsumeTraces(ctx, td)
	if err != nil {
		return &api.Status{
			Code:   api.StatusCodeError,
			Reason: err.Error(),
		}
	}

	return &api.Status{
		Code: api.StatusCodeSuccess,
	}
}

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

	// e, err := factory.CreateTraces(context.Background(), settings, &otlphttpexporter.Config{
	// 	ClientConfig: confighttp.ClientConfig{Endpoint: "http://localhost:4319"},
	// 	Encoding:     otlphttpexporter.EncodingProto,
	// })
	// if err != nil {
	// 	panic(err)
	// }

	// if err := e.Start(context.Background(), componenttest.NewNopHost()); err != nil {
	// 	panic(err)
	// }

	// plugin.Set(exp{
	// 	e: e,
	// })
}

func main() {}
