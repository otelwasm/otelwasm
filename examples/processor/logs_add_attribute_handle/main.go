// logs_add_attribute_handle adds a fixed attribute to every log record using
// the handle/accessor ABI: telemetry stays on the host and is mutated in
// place through fine-grained host functions, with no marshal/unmarshal round
// trip.
//
// This is the handle counterpart of logs_add_attribute_proto.
package main

import "github.com/otelwasm/otelwasm/guest/pdatahandle"

func main() {}

const telemetryTypeLogs = 1 << 1

//go:wasmexport getSupportedTelemetry
func getSupportedTelemetry() uint32 {
	return telemetryTypeLogs
}

//go:wasmexport processLogs
func processLogs() uint32 {
	logs := pdatahandle.CurrentLogs()
	rls := logs.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		sls := rls.At(i).ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			lrs := sls.At(j).LogRecords()
			for k := 0; k < lrs.Len(); k++ {
				lrs.At(k).Attributes().PutStr("example_key", "example_value")
			}
		}
	}
	pdatahandle.CommitLogs()
	return 0
}
