package main

import (
	"context"

	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/plugin" // register tracesreceiver
)

func init() {
	plugin.Set(&NopReceiver{})
}
func main() {}

var (
	// _ api.TracesReceiver  = (*NopReceiver)(nil)
	_ api.MetricsReceiver = (*NopReceiver)(nil)
	// _ api.LogsReceiver    = (*NopReceiver)(nil)
)

type NopReceiver struct{}

// StartTraces implements api.TracesReceiver.
func (n *NopReceiver) StartTraces(ctx context.Context) {
	<-ctx.Done()
}

// StartMetrics implements api.MetricsReceiver.
func (n *NopReceiver) StartMetrics(ctx context.Context) {
	<-ctx.Done()
}

// StartLogs implements api.LogsReceiver.
func (n *NopReceiver) StartLogs(ctx context.Context) {
	<-ctx.Done()
}
