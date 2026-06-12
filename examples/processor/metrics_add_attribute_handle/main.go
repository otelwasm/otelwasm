// metrics_add_attribute_handle adds a fixed attribute to every data point
// using the handle/accessor ABI: telemetry stays on the host and is mutated
// in place through fine-grained host functions, with no marshal/unmarshal
// round trip.
//
// This is the handle counterpart of metrics_add_attribute_proto.
package main

import (
	"github.com/otelwasm/otelwasm/guest/pdatahandle"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func main() {}

const telemetryTypeMetrics = 1 << 0

//go:wasmexport getSupportedTelemetry
func getSupportedTelemetry() uint32 {
	return telemetryTypeMetrics
}

//go:wasmexport processMetrics
func processMetrics() uint32 {
	metrics := pdatahandle.CurrentMetrics()
	rms := metrics.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		sms := rms.At(i).ScopeMetrics()
		for j := 0; j < sms.Len(); j++ {
			ms := sms.At(j).Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				switch m.Type() {
				case pmetric.MetricTypeGauge:
					dps := m.Gauge().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						dps.At(l).Attributes().PutStr("example_key", "example_value")
					}
				case pmetric.MetricTypeSum:
					dps := m.Sum().DataPoints()
					for l := 0; l < dps.Len(); l++ {
						dps.At(l).Attributes().PutStr("example_key", "example_value")
					}
				}
			}
		}
	}
	pdatahandle.CommitMetrics()
	return 0
}
