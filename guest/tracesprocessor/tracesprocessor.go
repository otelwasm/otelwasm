package tracesprocessor

import (
	"runtime"

	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/internal/imports"
	"github.com/musaprg/otelwasm/guest/internal/plugin"
)

var tracesprocessor api.TracesProcessor

func SetPlugin(tp api.TracesProcessor) {
	if tp == nil {
		return
	}
	tracesprocessor = tp
	plugin.MustSet(tp)
}

// go:wasmexport processTraces
func _processTraces() uint32 {
	traces := imports.CurrentTraces()
	result, status := tracesprocessor.ProcessTraces(traces)
	imports.SetResultTraces(result)
	runtime.KeepAlive(result) // until ptr is no longer needed

	return imports.StatusToCode(status)
}
