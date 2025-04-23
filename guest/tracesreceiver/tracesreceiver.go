package tracesreceiver

import (
	"context"
	"time"

	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/internal/imports"
	"github.com/musaprg/otelwasm/guest/internal/plugin"
)

var tracesreceiver api.TracesReceiver

func SetPlugin(mp api.TracesReceiver) {
	if mp == nil {
		panic("nil TracesReceiver")
	}
	tracesreceiver = mp
	plugin.MustSet(mp)
}

var _ func() = _startTracesReceiver

//go:wasmexport startTracesReceiver
func _startTracesReceiver() {
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

	tracesreceiver.StartTraces(ctx)
}
