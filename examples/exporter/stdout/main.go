package main

import (
	"fmt"

	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/plugin" // register tracesexporter, metricsexporter, logsexporter
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func init() {
	plugin.Set(&StdoutExporter{})
}
func main() {}

var (
	_ api.TracesExporter  = (*StdoutExporter)(nil)
	_ api.MetricsExporter = (*StdoutExporter)(nil)
	_ api.LogsExporter    = (*StdoutExporter)(nil)
)

type StdoutExporter struct{}

// PushTraces implements api.TracesExporter.
func (e *StdoutExporter) PushTraces(traces ptrace.Traces) *api.Status {
	marshaler := ptrace.JSONMarshaler{}
	jsonData, err := marshaler.MarshalTraces(traces)
	if err != nil {
		return &api.Status{
			Code:   api.StatusCodeError,
			Reason: fmt.Sprintf("failed to marshal traces to JSON: %v", err),
		}
	}

	fmt.Println(string(jsonData))
	return nil
}

// PushMetrics implements api.MetricsExporter.
func (e *StdoutExporter) PushMetrics(metrics pmetric.Metrics) *api.Status {
	marshaler := pmetric.JSONMarshaler{}
	jsonData, err := marshaler.MarshalMetrics(metrics)
	if err != nil {
		return &api.Status{
			Code:   api.StatusCodeError,
			Reason: fmt.Sprintf("failed to marshal metrics to JSON: %v", err),
		}
	}

	fmt.Println(string(jsonData))
	return nil
}

// PushLogs implements api.LogsExporter.
func (e *StdoutExporter) PushLogs(logs plog.Logs) *api.Status {
	marshaler := plog.JSONMarshaler{}
	jsonData, err := marshaler.MarshalLogs(logs)
	if err != nil {
		return &api.Status{
			Code:   api.StatusCodeError,
			Reason: fmt.Sprintf("failed to marshal logs to JSON: %v", err),
		}
	}

	fmt.Println(string(jsonData))
	return nil
}
