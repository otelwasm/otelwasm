// passthrough_proto reads the name and kind of every span and passes the
// data through unchanged, using the proto-serialization ABI. Even though
// nothing is mutated, the whole payload is marshaled/unmarshaled on both
// sides of the host/guest boundary.
//
// This is the baseline counterpart of passthrough_handle.
package main

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/plugin"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func init() {
	plugin.Set(&PassthroughProcessor{})
}

func main() {}

var _ api.TracesProcessor = (*PassthroughProcessor)(nil)

type PassthroughProcessor struct{}

// observed keeps the read results alive so the reads cannot be optimized out.
var observed int

func (p *PassthroughProcessor) ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *api.Status) {
	rss := traces.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		sss := rss.At(i).ScopeSpans()
		for j := 0; j < sss.Len(); j++ {
			spans := sss.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				observed += len(span.Name()) + int(span.Kind())
			}
		}
	}
	return traces, api.StatusSuccess()
}
