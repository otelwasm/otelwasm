package logsreceiver

import (
	"context"
	"time"

	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/internal/imports"
	"github.com/musaprg/otelwasm/guest/internal/plugin"
)

var logsreceiver api.LogsReceiver

func SetPlugin(mp api.LogsReceiver) {
	if mp == nil {
		panic("nil LogsReceiver")
	}
	logsreceiver = mp
	plugin.MustSet(mp)
}

var _ func() = _startLogsReceiver

//go:wasmexport startLogsReceiver
func _startLogsReceiver() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if imports.GetShutdownRequested() {
					cancel()
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	logsreceiver.StartLogs(ctx)
}
