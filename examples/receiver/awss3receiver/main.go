package main

import (
	"context"

	"github.com/go-viper/mapstructure/v2"
	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/imports"
	"github.com/musaprg/otelwasm/guest/plugin" // register receivers
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awss3receiver"
	_ "github.com/stealthrocket/net/http"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

var (
	otlpReceiver OTLPReceiver
	logger       *zap.Logger
)

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}

	plugin.Set(&otlpReceiver)

	telemetrySettings := componenttest.NewNopTelemetrySettings()
	telemetrySettings.Logger = logger
	otlpReceiver.factory = awss3receiver.NewFactory()

	settings := receiver.Settings{
		ID:                component.MustNewID("awss3"),
		TelemetrySettings: telemetrySettings,
		BuildInfo:         component.NewDefaultBuildInfo(),
	}

	otlpReceiver.settings = settings
}
func main() {}

var (
	_ api.LogsReceiver    = (*OTLPReceiver)(nil)
	_ api.MetricsReceiver = (*OTLPReceiver)(nil)
	_ api.TracesReceiver  = (*OTLPReceiver)(nil)
)

type OTLPReceiver struct {
	factory  receiver.Factory
	cfg      *awss3receiver.Config
	settings receiver.Settings

	guestConsumer *guestConsumer
}

func (n *OTLPReceiver) initConfig() {
	if n.cfg != nil {
		return
	}

	var config any
	err := imports.GetConfig(&config)
	if err != nil {
		logger.Fatal("failed to get config", zap.Error(err))
	}

	n.cfg = n.factory.CreateDefaultConfig().(*awss3receiver.Config)

	if err := mapstructure.Decode(config, &n.cfg); err != nil {
		logger.Fatal("failed to decode config", zap.Error(err))
	}
	n.settings.Logger.Debug("config", zap.Any("config", n.cfg))
}

// StartLogs implements api.LogsReceiver.
func (n *OTLPReceiver) StartLogs(ctx context.Context) {
	n.initConfig()

	logsConsumer, err := consumer.NewLogs(n.guestConsumer.ConsumeLogs, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
	if err != nil {
		logger.Fatal("failed to create logs consumer", zap.Error(err))
	}

	logs, err := n.factory.CreateLogs(ctx, n.settings, n.cfg, logsConsumer)
	if err != nil {
		logger.Fatal("failed to create logs receiver", zap.Error(err))
	}

	err = logs.Start(ctx, componenttest.NewNopHost())
	if err != nil {
		logger.Fatal("failed to start logs receiver", zap.Error(err))
	}

	<-ctx.Done()
	err = logs.Shutdown(ctx)
	if err != nil {
		logger.Fatal("failed to shutdown logs receiver", zap.Error(err))
	}
}

func (n *OTLPReceiver) StartMetrics(ctx context.Context) {
	n.initConfig()

	metricsConsumer, err := consumer.NewMetrics(n.guestConsumer.ConsumeMetrics, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
	if err != nil {
		logger.Fatal("failed to create metrics consumer", zap.Error(err))
	}

	metrics, err := n.factory.CreateMetrics(ctx, n.settings, n.cfg, metricsConsumer)
	if err != nil {
		logger.Fatal("failed to create metrics receiver", zap.Error(err))
	}

	err = metrics.Start(ctx, componenttest.NewNopHost())
	if err != nil {
		logger.Fatal("failed to start metrics receiver", zap.Error(err))
	}

	<-ctx.Done()
	err = metrics.Shutdown(ctx)
	if err != nil {
		logger.Fatal("failed to shutdown metrics receiver", zap.Error(err))
	}
}

func (n *OTLPReceiver) StartTraces(ctx context.Context) {
	n.initConfig()

	tracesConsumer, err := consumer.NewTraces(n.guestConsumer.ConsumeTraces, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
	if err != nil {
		logger.Fatal("failed to create traces consumer", zap.Error(err))
	}

	traces, err := n.factory.CreateTraces(ctx, n.settings, n.cfg, tracesConsumer)
	if err != nil {
		logger.Fatal("failed to create traces receiver", zap.Error(err))
	}

	err = traces.Start(ctx, componenttest.NewNopHost())
	if err != nil {
		logger.Fatal("failed to start traces receiver", zap.Error(err))
	}

	<-ctx.Done()
	err = traces.Shutdown(ctx)
	if err != nil {
		logger.Fatal("failed to shutdown traces receiver", zap.Error(err))
	}
}

type guestConsumer struct {
}

func (c *guestConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	imports.SetResultLogs(ld)
	return nil
}

func (c *guestConsumer) ConsumeMetrics(ctx context.Context, ld pmetric.Metrics) error {
	imports.SetResultMetrics(ld)
	return nil
}

func (c *guestConsumer) ConsumeTraces(ctx context.Context, ld ptrace.Traces) error {
	imports.SetResultTraces(ld)
	return nil
}
