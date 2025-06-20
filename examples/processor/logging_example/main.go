package main

import (
	"log/slog"

	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/logging"
	"github.com/otelwasm/otelwasm/guest/plugin"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type loggingProcessor struct{}

func (p *loggingProcessor) ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *api.Status) {
	logging.Info("Processing traces", map[string]string{
		"trace_count": string(rune(traces.SpanCount())),
		"component":   "logging_example_processor",
	})

	// Log details about each trace
	for i := 0; i < traces.ResourceSpans().Len(); i++ {
		rs := traces.ResourceSpans().At(i)
		
		logging.Debug("Processing resource spans", map[string]string{
			"resource_index": string(rune(i)),
			"scope_count":    string(rune(rs.ScopeSpans().Len())),
		})

		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)
			
			logging.Debug("Processing scope spans", map[string]string{
				"scope_index": string(rune(j)),
				"span_count":  string(rune(ss.Spans().Len())),
			})

			for k := 0; k < ss.Spans().Len(); k++ {
				span := ss.Spans().At(k)
				
				// Log each span with its details
				logger := logging.NewLogger()
				logger.LogAttrs(slog.LevelInfo, "Processing span",
					slog.String("span_name", span.Name()),
					slog.String("trace_id", span.TraceID().String()),
					slog.String("span_id", span.SpanID().String()),
				)
			}
		}
	}

	logging.Info("Finished processing traces", map[string]string{
		"total_spans": string(rune(traces.SpanCount())),
	})

	return traces, nil
}

func (p *loggingProcessor) ProcessMetrics(metrics pmetric.Metrics) (pmetric.Metrics, *api.Status) {
	logging.Info("Processing metrics", map[string]string{
		"metric_count": string(rune(metrics.MetricCount())),
		"component":    "logging_example_processor",
	})

	// Log details about each metric
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		rm := metrics.ResourceMetrics().At(i)
		
		logging.Debug("Processing resource metrics", map[string]string{
			"resource_index": string(rune(i)),
			"scope_count":    string(rune(rm.ScopeMetrics().Len())),
		})

		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			
			for k := 0; k < sm.Metrics().Len(); k++ {
				metric := sm.Metrics().At(k)
				
				logger := logging.NewLogger()
				logger.LogAttrs(slog.LevelInfo, "Processing metric",
					slog.String("metric_name", metric.Name()),
					slog.String("metric_type", metric.Type().String()),
				)
			}
		}
	}

	logging.Info("Finished processing metrics")

	return metrics, nil
}

func (p *loggingProcessor) ProcessLogs(logs plog.Logs) (plog.Logs, *api.Status) {
	logging.Info("Processing logs", map[string]string{
		"log_count": string(rune(logs.LogRecordCount())),
		"component": "logging_example_processor",
	})

	// Log details about each log record
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		rl := logs.ResourceLogs().At(i)
		
		logging.Debug("Processing resource logs", map[string]string{
			"resource_index": string(rune(i)),
			"scope_count":    string(rune(rl.ScopeLogs().Len())),
		})

		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			
			for k := 0; k < sl.LogRecords().Len(); k++ {
				logRecord := sl.LogRecords().At(k)
				
				logger := logging.NewLogger()
				logger.LogAttrs(slog.LevelInfo, "Processing log record",
					slog.String("log_body", logRecord.Body().AsString()),
					slog.String("severity", logRecord.SeverityText()),
				)
			}
		}
	}

	if logs.LogRecordCount() > 10 {
		logging.Warn("High log volume detected", map[string]string{
			"log_count": string(rune(logs.LogRecordCount())),
			"threshold": "10",
		})
	}

	logging.Info("Finished processing logs")

	return logs, nil
}

func init() {
	logging.Info("Initializing logging example processor")

	// Register the processor for all telemetry types
	plugin.Set(struct {
		api.TracesProcessor
		api.MetricsProcessor  
		api.LogsProcessor
	}{
		&loggingProcessor{},
		&loggingProcessor{},
		&loggingProcessor{},
	})

	logging.Info("Logging example processor initialized successfully")
}

func main() {}