// passthrough_handle reads the name and kind of every span and passes the
// data through unchanged, using the handle/accessor ABI. Only the fields
// actually read cross the host/guest boundary; the payload is never
// serialized.
//
// This is the handle counterpart of passthrough_proto.
package main

import "github.com/otelwasm/otelwasm/guest/pdatahandle"

func main() {}

const telemetryTypeTraces = 1 << 2

//go:wasmexport getSupportedTelemetry
func getSupportedTelemetry() uint32 {
	return telemetryTypeTraces
}

// observed keeps the read results alive so the reads cannot be optimized out.
var observed int

//go:wasmexport processTraces
func processTraces() uint32 {
	traces := pdatahandle.CurrentTraces()
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
	pdatahandle.Commit()
	return 0
}
