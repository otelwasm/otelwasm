package wasmreceiver

import (
	"context"
	"sync"

	"github.com/musaprg/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
)

type Receiver struct {
	cfg          *Config
	plugin       *wasmplugin.WasmPlugin
	nextConsumer consumer.Metrics

	stack *wasmplugin.Stack
	wg    sync.WaitGroup
}

func newWasmReceiver(ctx context.Context, cfg *Config, nextConsumer consumer.Metrics) (context.Context, *Receiver, error) {
	if err := cfg.Validate(); err != nil {
		return ctx, nil, err
	}

	// Specify required functions for the processor
	requiredFunctions := []string{"startMetricsReceiver"}

	// Create a wasmplugin configuration from our processor config
	pluginCfg := &wasmplugin.Config{
		Path:         cfg.Path,
		PluginConfig: cfg.PluginConfig,
	}

	// Initialize the WASM plugin
	ctx, plugin, err := wasmplugin.NewWasmPlugin(ctx, pluginCfg, requiredFunctions)
	if err != nil {
		return ctx, nil, err
	}

	return ctx, &Receiver{
		cfg:          cfg,
		plugin:       plugin,
		nextConsumer: nextConsumer,
	}, nil
}

var _ receiver.Metrics = (*Receiver)(nil)

// Start tells the component to start. Host parameter can be used for communicating
// with the host after Start() has already returned. If an error is returned by
// Start() then the collector startup will be aborted.
// If this is an exporter component it may prepare for exporting
// by connecting to the endpoint.
//
// If the component needs to perform a long-running starting operation then it is recommended
// that Start() returns quickly and the long-running operation is performed in background.
// In that case make sure that the long-running operation does not use the context passed
// to Start() function since that context will be cancelled soon and can abort the long-running
// operation. Create a new context from the context.Background() for long-running operations.
func (r *Receiver) Start(ctx context.Context, host component.Host) error {
	onResultMetricsChange := func(resultMetrics pmetric.Metrics) {
		r.nextConsumer.ConsumeMetrics(ctx, resultMetrics)
	}

	r.stack = &wasmplugin.Stack{
		OnResultMetricsChange: onResultMetricsChange,
		PluginConfigJSON:      r.plugin.PluginConfigJSON,
	}

	r.wg.Add(1)
	go r.runMetrics(ctx)

	return nil
}

func (r *Receiver) runMetrics(ctx context.Context) error {
	_, err := r.plugin.ProcessFunctionCall(ctx, "startMetricsReceiver", r.stack)
	if err != nil {
		return err
	}

	return nil
}

// Shutdown is invoked during service shutdown. After Shutdown() is called, if the component
// accepted data in any way, it should not accept it anymore.
//
// This method must be safe to call:
//   - without Start() having been called
//   - if the component is in a shutdown state already
//
// If there are any background operations running by the component they must be aborted before
// this function returns. Remember that if you started any long-running background operations from
// the Start() method, those operations must be also cancelled. If there are any buffers in the
// component, they should be cleared and the data sent immediately to the next component.
//
// The component's lifecycle is completed once the Shutdown() method returns. No other
// methods of the component are called after that. If necessary a new component with
// the same or different configuration may be created and started (this may happen
// for example if we want to restart the component).
func (r *Receiver) Shutdown(ctx context.Context) error {
	r.stack.RequestedShutdown.Store(true)
	// TODO: Set timeout for shutdown

	r.wg.Wait()

	return nil
}
