package processor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// Config defines the configuration for the add_new_attribute processor
type Config struct {
	AttributeName  string `mapstructure:"attribute_name"`
	AttributeValue string `mapstructure:"attribute_value"`
}

func createDefaultConfig() component.Config {
	return &Config{
		AttributeName:  "",
		AttributeValue: "",
	}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	processorConfig, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("configuration is not of type Config but %T", cfg)
	}

	if processorConfig.AttributeName == "" {
		return nil, fmt.Errorf("attribute_name is required")
	}
	if processorConfig.AttributeValue == "" {
		return nil, fmt.Errorf("attribute_value is required")
	}

	proc := &attributeProcessor{
		attributeName:  processorConfig.AttributeName,
		attributeValue: processorConfig.AttributeValue,
	}

	return processorhelper.NewTraces(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.processTraces,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	processorConfig, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("configuration is not of type Config but %T", cfg)
	}

	if processorConfig.AttributeName == "" {
		return nil, fmt.Errorf("attribute_name is required")
	}
	if processorConfig.AttributeValue == "" {
		return nil, fmt.Errorf("attribute_value is required")
	}

	proc := &attributeProcessor{
		attributeName:  processorConfig.AttributeName,
		attributeValue: processorConfig.AttributeValue,
	}

	return processorhelper.NewMetrics(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.processMetrics,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	processorConfig, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("configuration is not of type Config but %T", cfg)
	}

	if processorConfig.AttributeName == "" {
		return nil, fmt.Errorf("attribute_name is required")
	}
	if processorConfig.AttributeValue == "" {
		return nil, fmt.Errorf("attribute_value is required")
	}

	proc := &attributeProcessor{
		attributeName:  processorConfig.AttributeName,
		attributeValue: processorConfig.AttributeValue,
	}

	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.processLogs,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

// attributeProcessor implements the processor for adding a new attribute to telemetry data
type attributeProcessor struct {
	attributeName  string
	attributeValue string
}

// processTraces adds the configured attribute to all spans
func (p *attributeProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		scopeSpans := rs.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			ss := scopeSpans.At(j)
			spans := ss.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				span.Attributes().PutStr(p.attributeName, p.attributeValue)
			}
		}
	}
	return td, nil
}

// processMetrics adds the configured attribute to all metric data points
func (p *attributeProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		scopeMetrics := rm.ScopeMetrics()
		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)
			metrics := sm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				p.addAttributeToMetric(metric)
			}
		}
	}
	return md, nil
}

// addAttributeToMetric adds the configured attribute to different metric types
func (p *attributeProcessor) addAttributeToMetric(metric pmetric.Metric) {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dps.At(i).Attributes().PutStr(p.attributeName, p.attributeValue)
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dps.At(i).Attributes().PutStr(p.attributeName, p.attributeValue)
		}
	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dps.At(i).Attributes().PutStr(p.attributeName, p.attributeValue)
		}
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dps.At(i).Attributes().PutStr(p.attributeName, p.attributeValue)
		}
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dps.At(i).Attributes().PutStr(p.attributeName, p.attributeValue)
		}
	}
}

// processLogs adds the configured attribute to all log records
func (p *attributeProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		scopeLogs := rl.ScopeLogs()
		for j := 0; j < scopeLogs.Len(); j++ {
			sl := scopeLogs.At(j)
			logs := sl.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)
				log.Attributes().PutStr(p.attributeName, p.attributeValue)
			}
		}
	}
	return ld, nil
}

func NewFactory() processor.Factory {
	return processor.NewFactory(
		Type,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, TracesStability),
		processor.WithLogs(createLogsProcessor, LogsStability),
		processor.WithMetrics(createMetricsProcessor, MetricsStability))
}
