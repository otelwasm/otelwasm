package logsprocessor

import (
	"runtime"

	"github.com/otelwasm/otelwasm/guest/api"
	pubimports "github.com/otelwasm/otelwasm/guest/imports"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
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

var _ func() uint32 = _processLogs

//go:wasmexport processLogs
func _processLogs() uint32 {
	logs := imports.CurrentLogs()
	result, status := logsprocessor.ProcessLogs(logs)
	if (result == plog.Logs{}) {
		pubimports.SetResultLogs(result)
	}
	runtime.KeepAlive(result) // until ptr is no longer needed

	return imports.StatusToCode(status)
}
