package logsexporter

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/mem"
	"github.com/otelwasm/otelwasm/guest/internal/plugin"
	"go.opentelemetry.io/collector/pdata/plog"
)

var logsexporter api.LogsExporter

func SetPlugin(tp api.LogsExporter) {
	if tp == nil {
		panic("nil LogsExporter")
	}
	logsexporter = tp
	plugin.MustSet(tp)
}

var _ func(uint32, uint32) uint32 = _consumeLogs

//go:wasmexport otelwasm_consume_logs
func _consumeLogs(dataPtr uint32, dataSize uint32) uint32 {
	raw := mem.TakeOwnership(dataPtr, dataSize)
	unmarshaler := plog.ProtoUnmarshaler{}
	logs, err := unmarshaler.UnmarshalLogs(raw)
	if err != nil {
		return imports.StatusToCode(api.StatusError(err.Error()))
	}

	status := logsexporter.PushLogs(logs)
	return imports.StatusToCode(status)
}
