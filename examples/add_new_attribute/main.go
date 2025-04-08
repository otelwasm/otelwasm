package main

import (
	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/plugin" // register tracesprocessor
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func init() {
	plugin.Set(&AttributeProcessor{})
}
func main() {}

var _ api.TracesProcessor = (*AttributeProcessor)(nil)

type AttributeProcessor struct{}

// ProcessTraces implements api.TracesProcessor.
func (n *AttributeProcessor) ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *api.Status) {
	// Add new attribute to all spans
	newTraces := ptrace.NewTraces()
	traces.CopyTo(newTraces)
	spans := newTraces.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
	for i := 0; i < spans.Len(); i++ {
		spans.At(i).Attributes().PutStr("new-attribute", "new-value")
	}
	return newTraces, nil
}
