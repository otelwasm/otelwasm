// logs_add_attribute_proto adds a fixed attribute to every log record using
// the proto-serialization ABI: the whole payload is marshaled/unmarshaled on
// both sides of the host/guest boundary.
//
// This is the baseline counterpart of logs_add_attribute_handle.
package main

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/plugin"
	"go.opentelemetry.io/collector/pdata/plog"
)

func init() {
	plugin.Set(&AttributeProcessor{})
}

func main() {}

var _ api.LogsProcessor = (*AttributeProcessor)(nil)

type AttributeProcessor struct{}

func (p *AttributeProcessor) ProcessLogs(logs plog.Logs) (plog.Logs, *api.Status) {
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
	return logs, api.StatusSuccess()
}
