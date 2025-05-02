package tracesprocessor

import (
	"runtime"

	"github.com/otelwasm/otelwasm/guest/api"
	pubimports "github.com/otelwasm/otelwasm/guest/imports"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/plugin"
)

var tracesprocessor api.TracesProcessor

func SetPlugin(tp api.TracesProcessor) {
	if tp == nil {
		panic("nil TracesProcessor")
	}
	tracesprocessor = tp
	plugin.MustSet(tp)
}

var _ func() uint32 = _processTraces

//go:wasmexport processTraces
func _processTraces() uint32 {
	traces := imports.CurrentTraces()
	result, status := tracesprocessor.ProcessTraces(traces)
	pubimports.SetResultTraces(result)
	runtime.KeepAlive(result) // until ptr is no longer needed

	return imports.StatusToCode(status)
}
