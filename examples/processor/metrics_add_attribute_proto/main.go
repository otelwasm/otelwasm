// metrics_add_attribute_proto adds a fixed attribute to every data point
// using the proto-serialization ABI: the whole payload is
// marshaled/unmarshaled on both sides of the host/guest boundary.
//
// This is the baseline counterpart of metrics_add_attribute_handle.
package main

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/plugin"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func init() {
	plugin.Set(&AttributeProcessor{})
}

func main() {}

var _ api.MetricsProcessor = (*AttributeProcessor)(nil)

type AttributeProcessor struct{}

func (p *AttributeProcessor) ProcessMetrics(metrics pmetric.Metrics) (pmetric.Metrics, *api.Status) {
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
	return metrics, api.StatusSuccess()
}
