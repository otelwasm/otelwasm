package main

import (
	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/plugin" // register tracesprocessor
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
