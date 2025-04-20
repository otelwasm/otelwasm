package main

import (
	"context"

	"github.com/musaprg/otelwasm/examples/receiver/webhookeventreceiver/webhookeventreceiver"
	"github.com/musaprg/otelwasm/guest/api"
	otelwasm "github.com/musaprg/otelwasm/guest/imports"
	"github.com/musaprg/otelwasm/guest/plugin" // register receivers
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

func init() {
	println("registering webhookeventreceiver")
	plugin.Set(&WebhookEventReceiver{})
}
func main() {}

var (
	_ api.LogsReceiver = (*WebhookEventReceiver)(nil)
)

type logConsumer struct{}

// Capabilities implements consumer.Logs.
func (c *logConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

var _ consumer.Logs = (*logConsumer)(nil)

func (c *logConsumer) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	return otelwasm.SetResultLogs(logs)
}

type host struct {
}

func (h *host) GetExtensions() map[component.ID]component.Component {
	return nil
}

type WebhookEventReceiver struct{}

// StartLogs implements api.LogsReceiver.
func (n *WebhookEventReceiver) StartLogs(ctx context.Context) {
	println("called startlogs")

	cfg := webhookeventreceiver.CreateDefaultConfig().(*webhookeventreceiver.Config)
	cfg.Endpoint = "localhost:8088"
	csm := &logConsumer{}
	lr, err := webhookeventreceiver.NewLogsReceiver(*cfg, csm)
	if err != nil {
		panic(err)
	}

	println("initialization completed")

	lr.Start(ctx, &host{})
	<-ctx.Done()
	println("stopping receiver")
}
