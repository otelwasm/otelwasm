package main

import (
	"context"

	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/factoryconnector"
	"github.com/otelwasm/otelwasm/guest/imports"
	"github.com/otelwasm/otelwasm/guest/plugin" // register processors
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

func init() {
	// Create the factory
	factory := &LoggingProcessorFactory{}
	
	settings := processor.Settings{
		ID:                component.MustNewID("logging"),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}

	connector := factoryconnector.NewProcessorConnector[component.Config](
		factory,
		settings,
	)

	// Register the processor
	plugin.RegisterProcessor(connector)
}

func main() {}

// LoggingProcessorFactory creates LoggingProcessor instances
type LoggingProcessorFactory struct{}

func (f *LoggingProcessorFactory) Type() component.Type {
	return component.MustNewType("logging")
}

func (f *LoggingProcessorFactory) CreateDefaultConfig() component.Config {
	return &LoggingProcessorConfig{}
}

func (f *LoggingProcessorFactory) CreateTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	return &LoggingProcessor{nextConsumer: nextConsumer}, nil
}

func (f *LoggingProcessorFactory) CreateMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	return nil, component.ErrDataTypeIsNotSupported
}

func (f *LoggingProcessorFactory) CreateLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	return nil, component.ErrDataTypeIsNotSupported
}

// LoggingProcessorConfig holds the configuration for the logging processor
type LoggingProcessorConfig struct{}

// LoggingProcessor demonstrates structured logging functionality
type LoggingProcessor struct {
	nextConsumer consumer.Traces
}

func (p *LoggingProcessor) Start(ctx context.Context, host component.Host) error {
	imports.LogInfo("Logging processor started", map[string]interface{}{
		"component": "logging_example_processor",
		"version":   "1.0.0",
	})
	return nil
}

func (p *LoggingProcessor) Shutdown(ctx context.Context) error {
	imports.LogInfo("Logging processor shutdown", map[string]interface{}{
		"component": "logging_example_processor",
	})
	return nil
}

func (p *LoggingProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *LoggingProcessor) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	imports.LogInfo("Starting trace processing", map[string]interface{}{
		"component": "logging_example_processor",
		"version":   "1.0.0",
	})

	imports.LogDebug("Received traces for processing", map[string]interface{}{
		"span_count":     traces.SpanCount(),
		"resource_count": traces.ResourceSpans().Len(),
	})

	// Process each resource span
	for i := 0; i < traces.ResourceSpans().Len(); i++ {
		rs := traces.ResourceSpans().At(i)
		
		imports.LogDebug("Processing resource span", map[string]interface{}{
			"resource_index": i,
			"scope_count":    rs.ScopeSpans().Len(),
		})

		// Process each scope span
		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)
			
			imports.LogDebug("Processing scope span", map[string]interface{}{
				"scope_index": j,
				"span_count":  ss.Spans().Len(),
			})

			// Process each span
			for k := 0; k < ss.Spans().Len(); k++ {
				span := ss.Spans().At(k)
				
				// Log span details
				imports.LogInfo("Processing span", map[string]interface{}{
					"span_id":    span.SpanID().String(),
					"trace_id":   span.TraceID().String(),
					"span_name":  span.Name(),
					"span_kind":  span.Kind().String(),
					"start_time": span.StartTimestamp().AsTime().String(),
					"end_time":   span.EndTimestamp().AsTime().String(),
				})

				// Add a custom attribute to demonstrate processing
				span.Attributes().PutStr("processed_by", "logging_example_processor")
				
				// Log attribute addition
				imports.LogDebug("Added processing attribute", map[string]interface{}{
					"span_id":   span.SpanID().String(),
					"attribute": "processed_by",
					"value":     "logging_example_processor",
				})

				// Simulate some processing logic
				if span.Status().Code() == ptrace.StatusCodeError {
					imports.LogWarn("Found span with error status", map[string]interface{}{
						"span_id":        span.SpanID().String(),
						"error_message":  span.Status().Message(),
						"span_name":      span.Name(),
					})
				}
			}
		}
	}

	imports.LogInfo("Finished trace processing", map[string]interface{}{
		"component":   "logging_example_processor",
		"spans_total": traces.SpanCount(),
	})

	return p.nextConsumer.ConsumeTraces(ctx, traces)
}