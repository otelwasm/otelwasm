package tracesexporter

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/plugin"
)

var tracesexporter api.TracesExporter

func SetPlugin(tp api.TracesExporter) {
	if tp == nil {
		panic("nil TracesExporter")
	}
	tracesexporter = tp
	plugin.MustSet(tp)
}

var _ func() uint32 = _pushTraces

//go:wasmexport pushTraces
func _pushTraces() uint32 {
	traces := imports.CurrentTraces()
	status := tracesexporter.PushTraces(traces)
	return imports.StatusToCode(status)
}
