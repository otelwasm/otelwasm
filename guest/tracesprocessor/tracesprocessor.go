package tracesprocessor

import (
	"runtime"

	"github.com/otelwasm/otelwasm/guest/api"
	pubimports "github.com/otelwasm/otelwasm/guest/imports"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/mem"
	"github.com/otelwasm/otelwasm/guest/internal/plugin"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var tracesprocessor api.TracesProcessor

func SetPlugin(tp api.TracesProcessor) {
	if tp == nil {
		panic("nil TracesProcessor")
	}
	tracesprocessor = tp
	plugin.MustSet(tp)
}

var _ func(uint32, uint32) uint32 = _consumeTraces

//go:wasmexport otelwasm_consume_traces
func _consumeTraces(dataPtr uint32, dataSize uint32) uint32 {
	raw := mem.TakeOwnership(dataPtr, dataSize)
	unmarshaler := ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(raw)
	if err != nil {
		return imports.StatusToCode(api.StatusError(err.Error()))
	}

	result, status := tracesprocessor.ProcessTraces(traces)
	if result != (ptrace.Traces{}) {
		pubimports.SetResultTraces(result)
	}
	runtime.KeepAlive(result) // until ptr is no longer needed.
	return imports.StatusToCode(status)
}
