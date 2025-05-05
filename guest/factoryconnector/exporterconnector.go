// filepath: /Users/musaprg/workspace/personal/otelwasm/guest/factoryconnector/exporterconnector.go
package factoryconnector

import (
	"context"

	"github.com/go-viper/mapstructure/v2"
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/imports"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type ExporterConnector struct {
	factory  exporter.Factory
	cfg      component.Config
	settings exporter.Settings
}

func NewExporterConnector(
	factory exporter.Factory,
	settings exporter.Settings,
) *ExporterConnector {
	return &ExporterConnector{
		factory:  factory,
		settings: settings,
	}
}

func (e *ExporterConnector) Metrics() api.MetricsExporter {
	return &metricsExporter{ExporterConnector: e}
}

func (e *ExporterConnector) Logs() api.LogsExporter {
	return &logsExporter{ExporterConnector: e}
}

func (e *ExporterConnector) Traces() api.TracesExporter {
	return &tracesExporter{ExporterConnector: e}
}

func (e *ExporterConnector) initConfig() {
	if e.cfg != nil {
		return
	}
	logger := e.settings.Logger

	var config any
	err := imports.GetConfig(&config)
	if err != nil {
		logger.Fatal("failed to get config", zap.Error(err))
	}

	e.cfg = e.factory.CreateDefaultConfig()

	if err := mapstructure.Decode(config, &e.cfg); err != nil {
		logger.Fatal("failed to decode config", zap.Error(err))
	}
	logger.Debug("config", zap.Any("config", e.cfg))
}

type metricsExporter struct {
	*ExporterConnector
	metricsExporter exporter.Metrics
}

func (e *metricsExporter) PushMetrics(metrics pmetric.Metrics) *api.Status {
	if e.metricsExporter == nil {
		e.initConfig()
		logger := e.settings.Logger

		var err error
		e.metricsExporter, err = e.factory.CreateMetrics(context.Background(), e.settings, e.cfg)
		if err != nil {
			logger.Error("failed to create metrics exporter", zap.Error(err))
			return api.StatusError(err.Error())
		}

		err = e.metricsExporter.Start(context.Background(), componenttest.NewNopHost())
		if err != nil {
			logger.Error("failed to start metrics exporter", zap.Error(err))
			return api.StatusError(err.Error())
		}
	}

	err := e.metricsExporter.ConsumeMetrics(context.Background(), metrics)
	if err != nil {
		e.settings.Logger.Error("failed to export metrics", zap.Error(err))
		return api.StatusError(err.Error())
	}

	return api.StatusSuccess()
}

type logsExporter struct {
	*ExporterConnector
	logsExporter exporter.Logs
}

func (e *logsExporter) PushLogs(logs plog.Logs) *api.Status {
	if e.logsExporter == nil {
		e.initConfig()
		logger := e.settings.Logger

		var err error
		e.logsExporter, err = e.factory.CreateLogs(context.Background(), e.settings, e.cfg)
		if err != nil {
			logger.Error("failed to create logs exporter", zap.Error(err))
			return api.StatusError(err.Error())
		}

		err = e.logsExporter.Start(context.Background(), componenttest.NewNopHost())
		if err != nil {
			logger.Error("failed to start logs exporter", zap.Error(err))
			return api.StatusError(err.Error())
		}
	}

	err := e.logsExporter.ConsumeLogs(context.Background(), logs)
	if err != nil {
		e.settings.Logger.Error("failed to export logs", zap.Error(err))
		return api.StatusError(err.Error())
	}

	return api.StatusSuccess()
}

type tracesExporter struct {
	*ExporterConnector
	tracesExporter exporter.Traces
}

func (e *tracesExporter) PushTraces(traces ptrace.Traces) *api.Status {
	if e.tracesExporter == nil {
		e.initConfig()
		logger := e.settings.Logger

		var err error
		e.tracesExporter, err = e.factory.CreateTraces(context.Background(), e.settings, e.cfg)
		if err != nil {
			logger.Error("failed to create traces exporter", zap.Error(err))
			return api.StatusError(err.Error())
		}

		err = e.tracesExporter.Start(context.Background(), componenttest.NewNopHost())
		if err != nil {
			logger.Error("failed to start traces exporter", zap.Error(err))
			return api.StatusError(err.Error())
		}
	}

	err := e.tracesExporter.ConsumeTraces(context.Background(), traces)
	if err != nil {
		e.settings.Logger.Error("failed to export traces", zap.Error(err))
		return api.StatusError(err.Error())
	}

	return api.StatusSuccess()
}
