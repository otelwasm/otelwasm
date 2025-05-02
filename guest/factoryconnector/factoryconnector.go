package factoryconnector

import (
	"context"

	"github.com/go-viper/mapstructure/v2"
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/imports"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type ReceiverConnector struct {
	factory  receiver.Factory
	cfg      component.Config
	settings receiver.Settings
}

func NewReceiverConnector(
	factory receiver.Factory,
	settings receiver.Settings,
) *ReceiverConnector {
	return &ReceiverConnector{
		factory:  factory,
		settings: settings,
	}
}

func (n *ReceiverConnector) Metrics() api.MetricsReceiver {
	return &metricsReceiver{ReceiverConnector: n}
}

func (n *ReceiverConnector) Logs() api.LogsReceiver {
	return &logsReceiver{ReceiverConnector: n}
}

func (n *ReceiverConnector) Traces() api.TracesReceiver {
	return &tracesReceiver{ReceiverConnector: n}
}

func (n *ReceiverConnector) initConfig() {
	if n.cfg != nil {
		return
	}
	logger := n.settings.Logger

	var config any
	err := imports.GetConfig(&config)
	if err != nil {
		logger.Fatal("failed to get config", zap.Error(err))
	}

	n.cfg = n.factory.CreateDefaultConfig()

	if err := mapstructure.Decode(config, &n.cfg); err != nil {
		logger.Fatal("failed to decode config", zap.Error(err))
	}
	logger.Debug("config", zap.Any("config", n.cfg))
}

type metricsReceiver struct {
	*ReceiverConnector
}

func (n *metricsReceiver) StartMetrics(ctx context.Context) {
	n.initConfig()
	logger := n.settings.Logger

	metricsConsumer, err := consumer.NewMetrics(ConsumeMetrics, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
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

type logsReceiver struct {
	*ReceiverConnector
}

func (n *logsReceiver) StartLogs(ctx context.Context) {
	n.initConfig()
	logger := n.settings.Logger

	logsConsumer, err := consumer.NewLogs(ConsumeLogs, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
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

type tracesReceiver struct {
	*ReceiverConnector
}

func (n *tracesReceiver) StartTraces(ctx context.Context) {
	n.initConfig()
	logger := n.settings.Logger

	tracesConsumer, err := consumer.NewTraces(ConsumeTraces, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
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
