package main

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/plugin" // register tracesexporter, metricsexporter, logsexporter
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func init() {
	plugin.Set(&NopExporter{})
}
func main() {}

var (
	_ api.TracesExporter  = (*NopExporter)(nil)
	_ api.MetricsExporter = (*NopExporter)(nil)
	_ api.LogsExporter    = (*NopExporter)(nil)
)

type NopExporter struct{}

// PushTraces implements api.TracesExporter.
func (n *NopExporter) PushTraces(traces ptrace.Traces) *api.Status {
	// This is a no-op exporter, so we just return success without doing anything
	return nil
}

// PushMetrics implements api.MetricsExporter.
func (n *NopExporter) PushMetrics(metrics pmetric.Metrics) *api.Status {
	// This is a no-op exporter, so we just return success without doing anything
	return nil
}

// PushLogs implements api.LogsExporter.
func (n *NopExporter) PushLogs(logs plog.Logs) *api.Status {
	// This is a no-op exporter, so we just return success without doing anything
	return nil
}
