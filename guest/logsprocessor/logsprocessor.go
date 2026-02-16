package logsprocessor

import (
	"runtime"

	"github.com/otelwasm/otelwasm/guest/api"
	pubimports "github.com/otelwasm/otelwasm/guest/imports"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/mem"
	"github.com/otelwasm/otelwasm/guest/internal/plugin"
	"go.opentelemetry.io/collector/pdata/plog"
)

var logsprocessor api.LogsProcessor

func SetPlugin(lp api.LogsProcessor) {
	if lp == nil {
		panic("nil LogsProcessor")
	}
	logsprocessor = lp
	plugin.MustSet(lp)
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

	result, status := logsprocessor.ProcessLogs(logs)
	// If the result is not empty, set it in the host.
	// In case of empty result, the result should be written inside the guest call.
	if (result != plog.Logs{}) {
		pubimports.SetResultLogs(result)
	}
	runtime.KeepAlive(result) // until ptr is no longer needed

	return imports.StatusToCode(status)
}
