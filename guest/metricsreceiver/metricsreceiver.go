package metricsreceiver

import (
	"context"
	"time"

	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/internal/imports"
	"github.com/musaprg/otelwasm/guest/internal/plugin"
)

var metricsreceiver api.MetricsReceiver

func SetPlugin(mp api.MetricsReceiver) {
	if mp == nil {
		panic("nil MetricsReceiver")
	}
	metricsreceiver = mp
	plugin.MustSet(mp)
}

var _ func() = _startMetricsReceiver

//go:wasmexport startMetricsReceiver
func _startMetricsReceiver() {
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

	metricsreceiver.StartMetrics(ctx)
}
