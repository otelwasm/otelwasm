package tracesexporter

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/mem"
	"github.com/otelwasm/otelwasm/guest/internal/plugin"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var tracesexporter api.TracesExporter

func SetPlugin(tp api.TracesExporter) {
	if tp == nil {
		panic("nil TracesExporter")
	}
	tracesexporter = tp
	plugin.MustSet(tp)
}

var _ func(uint32, uint32) uint32 = _consumeTraces

//go:wasmexport consume_traces
func _consumeTraces(dataPtr uint32, dataSize uint32) uint32 {
	raw := mem.TakeOwnership(dataPtr, dataSize)
	unmarshaler := ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(raw)
	if err != nil {
		return imports.StatusToCode(api.StatusError(err.Error()))
	}

	status := tracesexporter.PushTraces(traces)
	return imports.StatusToCode(status)
}
