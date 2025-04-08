package main

import (
	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/plugin" // register tracesprocessor
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func init() {
	plugin.Set(&NopProcessor{})
}
func main() {}

var _ api.TracesProcessor = (*NopProcessor)(nil)

type NopProcessor struct{}

// ProcessTraces implements api.TracesProcessor.
func (n *NopProcessor) ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *api.Status) {
	return traces, nil
}
