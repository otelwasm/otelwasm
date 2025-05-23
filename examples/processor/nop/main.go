package main

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/plugin" // register tracesprocessor
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func init() {
	plugin.Set(&NopProcessor{})
}
func main() {}

var (
	_ api.TracesProcessor  = (*NopProcessor)(nil)
	_ api.MetricsProcessor = (*NopProcessor)(nil)
	_ api.LogsProcessor    = (*NopProcessor)(nil)
)

type NopProcessor struct{}

// ProcessTraces implements api.TracesProcessor.
func (n *NopProcessor) ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *api.Status) {
	return traces, nil
}

// ProcessMetrics implements api.MetricsProcessor.
func (n *NopProcessor) ProcessMetrics(metrics pmetric.Metrics) (pmetric.Metrics, *api.Status) {
	return metrics, nil
}

// ProcessLogs implements api.LogsProcessor.
func (n *NopProcessor) ProcessLogs(logs plog.Logs) (plog.Logs, *api.Status) {
	return logs, nil
}
