// add_attribute_handle adds a fixed attribute to every span using the
// handle/accessor ABI: telemetry stays on the host and is mutated in place
// through fine-grained host functions, with no marshal/unmarshal round trip.
//
// This is the handle counterpart of add_attribute_proto.
package main

import "github.com/otelwasm/otelwasm/guest/pdatahandle"

func main() {}

const telemetryTypeTraces = 1 << 2

//go:wasmexport getSupportedTelemetry
func getSupportedTelemetry() uint32 {
	return telemetryTypeTraces
}

//go:wasmexport processTraces
func processTraces() uint32 {
	traces := pdatahandle.CurrentTraces()
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
	pdatahandle.Commit()
	return 0
}
