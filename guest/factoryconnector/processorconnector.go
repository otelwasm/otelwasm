// filepath: /Users/musaprg/workspace/personal/otelwasm/guest/factoryconnector/processorconnector.go
package factoryconnector

import (
	"context"

	"github.com/go-viper/mapstructure/v2"
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/imports"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

type ProcessorConnector struct {
	factory  processor.Factory
	cfg      component.Config
	settings processor.Settings
}

func NewProcessorConnector(
	factory processor.Factory,
	settings processor.Settings,
) *ProcessorConnector {
	return &ProcessorConnector{
		factory:  factory,
		settings: settings,
	}
}

func (p *ProcessorConnector) Metrics() api.MetricsProcessor {
	return &metricsProcessor{ProcessorConnector: p}
}

func (p *ProcessorConnector) Logs() api.LogsProcessor {
	return &logsProcessor{ProcessorConnector: p}
}

func (p *ProcessorConnector) Traces() api.TracesProcessor {
	return &tracesProcessor{ProcessorConnector: p}
}

func (p *ProcessorConnector) initConfig() {
	if p.cfg != nil {
		return
	}
	logger := p.settings.Logger

	var config any
	err := imports.GetConfig(&config)
	if err != nil {
		logger.Fatal("failed to get config", zap.Error(err))
	}

	p.cfg = p.factory.CreateDefaultConfig()

	if err := mapstructure.Decode(config, &p.cfg); err != nil {
		logger.Fatal("failed to decode config", zap.Error(err))
	}
	logger.Debug("config", zap.Any("config", p.cfg))
}

type metricsProcessor struct {
	*ProcessorConnector
	metricsProcessor processor.Metrics
	nextConsumer     consumer.Metrics
}

func (p *metricsProcessor) ProcessMetrics(metrics pmetric.Metrics) (pmetric.Metrics, *api.Status) {
	if p.metricsProcessor == nil {
		p.initConfig()
		logger := p.settings.Logger

		// Create a consumer that will capture the processed results
		var err error
		p.nextConsumer, err = consumer.NewMetrics(ConsumeMetrics, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
		if err != nil {
			logger.Error("failed to create metrics consumer", zap.Error(err))
			return metrics, api.StatusError(err.Error())
		}

		// Create the processor with our consumer
		p.metricsProcessor, err = p.factory.CreateMetrics(context.Background(), p.settings, p.cfg, p.nextConsumer)
		if err != nil {
			logger.Error("failed to create metrics processor", zap.Error(err))
			return metrics, api.StatusError(err.Error())
		}

		// Start the processor
		err = p.metricsProcessor.Start(context.Background(), componenttest.NewNopHost())
		if err != nil {
			logger.Error("failed to start metrics processor", zap.Error(err))
			return metrics, api.StatusError(err.Error())
		}
	}

	// Process the metrics
	err := p.metricsProcessor.ConsumeMetrics(context.Background(), metrics)
	if err != nil {
		p.settings.Logger.Error("failed to process metrics", zap.Error(err))
		return metrics, api.StatusError(err.Error())
	}

	// Return empty metrics to indicate that the result was already written to memory
	return pmetric.Metrics{}, api.StatusSuccess()
}

type logsProcessor struct {
	*ProcessorConnector
	logsProcessor processor.Logs
	nextConsumer  consumer.Logs
}

func (p *logsProcessor) ProcessLogs(logs plog.Logs) (plog.Logs, *api.Status) {
	if p.logsProcessor == nil {
		p.initConfig()
		logger := p.settings.Logger

		// Create a consumer that will capture the processed results
		var err error
		p.nextConsumer, err = consumer.NewLogs(ConsumeLogs, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
		if err != nil {
			logger.Error("failed to create logs consumer", zap.Error(err))
			return logs, api.StatusError(err.Error())
		}

		// Create the processor with our consumer
		p.logsProcessor, err = p.factory.CreateLogs(context.Background(), p.settings, p.cfg, p.nextConsumer)
		if err != nil {
			logger.Error("failed to create logs processor", zap.Error(err))
			return logs, api.StatusError(err.Error())
		}

		// Start the processor
		err = p.logsProcessor.Start(context.Background(), componenttest.NewNopHost())
		if err != nil {
			logger.Error("failed to start logs processor", zap.Error(err))
			return logs, api.StatusError(err.Error())
		}
	}

	// Process the logs
	err := p.logsProcessor.ConsumeLogs(context.Background(), logs)
	if err != nil {
		p.settings.Logger.Error("failed to process logs", zap.Error(err))
		return logs, api.StatusError(err.Error())
	}

	// Return empty logs to indicate that the result was already written to memory
	return plog.Logs{}, api.StatusSuccess()
}

type tracesProcessor struct {
	*ProcessorConnector
	tracesProcessor processor.Traces
	nextConsumer    consumer.Traces
}

func (p *tracesProcessor) ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *api.Status) {
	if p.tracesProcessor == nil {
		p.initConfig()
		logger := p.settings.Logger

		// Create a consumer that will capture the processed results
		var err error
		p.nextConsumer, err = consumer.NewTraces(ConsumeTraces, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
		if err != nil {
			logger.Error("failed to create traces consumer", zap.Error(err))
			return traces, api.StatusError(err.Error())
		}

		// Create the processor with our consumer
		p.tracesProcessor, err = p.factory.CreateTraces(context.Background(), p.settings, p.cfg, p.nextConsumer)
		if err != nil {
			logger.Error("failed to create traces processor", zap.Error(err))
			return traces, api.StatusError(err.Error())
		}

		// Start the processor
		err = p.tracesProcessor.Start(context.Background(), componenttest.NewNopHost())
		if err != nil {
			logger.Error("failed to start traces processor", zap.Error(err))
			return traces, api.StatusError(err.Error())
		}
	}

	// Process the traces
	err := p.tracesProcessor.ConsumeTraces(context.Background(), traces)
	if err != nil {
		p.settings.Logger.Error("failed to process traces", zap.Error(err))
		return traces, api.StatusError(err.Error())
	}

	// Return empty traces to indicate that the result was already written to memory
	return ptrace.Traces{}, api.StatusSuccess()
}
