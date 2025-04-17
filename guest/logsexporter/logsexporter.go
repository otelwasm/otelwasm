package logsexporter

import (
	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/internal/imports"
	"github.com/musaprg/otelwasm/guest/internal/plugin"
)

var logsexporter api.LogsExporter

func SetPlugin(tp api.LogsExporter) {
	if tp == nil {
		panic("nil LogsExporter")
	}
	logsexporter = tp
	plugin.MustSet(tp)
}

var _ func() uint32 = _pushLogs

//go:wasmexport pushLogs
func _pushLogs() uint32 {
	logs := imports.CurrentLogs()
	status := logsexporter.PushLogs(logs)
	return imports.StatusToCode(status)
}
