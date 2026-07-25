// add_attribute_proto adds a fixed attribute to every span using the
// proto-serialization ABI: the whole payload is marshaled/unmarshaled on
// both sides of the host/guest boundary.
//
// This is the baseline counterpart of add_attribute_handle.
package main

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/plugin"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func init() {
	plugin.Set(&AttributeProcessor{})
}

func main() {}

var _ api.TracesProcessor = (*AttributeProcessor)(nil)

type AttributeProcessor struct{}

func (p *AttributeProcessor) ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *api.Status) {
	rss := traces.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		sss := rss.At(i).ScopeSpans()
		for j := 0; j < sss.Len(); j++ {
			spans := sss.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				spans.At(k).Attributes().PutStr("example_key", "example_value")
			}
		}
	}
	return traces, api.StatusSuccess()
}
