package wasmreceiver

import (
	"context"
	"fmt"
	"sync"

	"github.com/musaprg/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
)

type Receiver struct {
	cfg           *Config
	plugin        *wasmplugin.WasmPlugin
	nextConsumerM consumer.Metrics
	nextConsumerL consumer.Logs
	nextConsumerT consumer.Traces

	stack *wasmplugin.Stack
	wg    sync.WaitGroup
}

func newMetricsWasmReceiver(ctx context.Context, cfg *Config, nextConsumerM consumer.Metrics) (context.Context, *Receiver, error) {
	if err := cfg.Validate(); err != nil {
		return ctx, nil, err
	}

	requiredFunctions := []string{"startMetricsReceiver"}

	pluginCfg := &wasmplugin.Config{
		Path:         cfg.Path,
		PluginConfig: cfg.PluginConfig,
	}

	ctx, plugin, err := wasmplugin.NewWasmPlugin(ctx, pluginCfg, requiredFunctions)
	if err != nil {
		return ctx, nil, err
	}

	return ctx, &Receiver{
		cfg:           cfg,
		plugin:        plugin,
		nextConsumerM: nextConsumerM,
	}, nil
}

func newLogsWasmReceiver(ctx context.Context, cfg *Config, nextConsumerL consumer.Logs) (context.Context, *Receiver, error) {
	fmt.Println("newLogsWasmReceiver called")
	if err := cfg.Validate(); err != nil {
		return ctx, nil, err
	}

	requiredFunctions := []string{"startLogsReceiver"}

	pluginCfg := &wasmplugin.Config{
		Path:         cfg.Path,
		PluginConfig: cfg.PluginConfig,
	}

	ctx, plugin, err := wasmplugin.NewWasmPlugin(ctx, pluginCfg, requiredFunctions)
	if err != nil {
		return ctx, nil, err
	}

	return ctx, &Receiver{
		cfg:           cfg,
		plugin:        plugin,
		nextConsumerL: nextConsumerL,
	}, nil
}

func newTracesWasmReceiver(ctx context.Context, cfg *Config, nextConsumerT consumer.Traces) (context.Context, *Receiver, error) {
	if err := cfg.Validate(); err != nil {
		return ctx, nil, err
	}

	requiredFunctions := []string{"startTracesReceiver"}

	pluginCfg := &wasmplugin.Config{
		Path:         cfg.Path,
		PluginConfig: cfg.PluginConfig,
	}

	ctx, plugin, err := wasmplugin.NewWasmPlugin(ctx, pluginCfg, requiredFunctions)
	if err != nil {
		return ctx, nil, err
	}

	return ctx, &Receiver{
		cfg:           cfg,
		plugin:        plugin,
		nextConsumerT: nextConsumerT,
	}, nil
}

var _ receiver.Metrics = (*Receiver)(nil)
var _ receiver.Logs = (*Receiver)(nil)
var _ receiver.Traces = (*Receiver)(nil)

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
	fmt.Println("Start called")
	onResultMetricsChange := func(resultMetrics pmetric.Metrics) {
		if r.nextConsumerM != nil {
			r.nextConsumerM.ConsumeMetrics(ctx, resultMetrics)
		}
	}

	onResultLogsChange := func(resultLogs plog.Logs) {
		if r.nextConsumerL != nil {
			r.nextConsumerL.ConsumeLogs(ctx, resultLogs)
		}
	}

	onResultTracesChange := func(resultTraces ptrace.Traces) {
		if r.nextConsumerT != nil {
			r.nextConsumerT.ConsumeTraces(ctx, resultTraces)
		}
	}

	r.stack = &wasmplugin.Stack{
		OnResultMetricsChange: onResultMetricsChange,
		OnResultLogsChange:    onResultLogsChange,
		OnResultTracesChange:  onResultTracesChange,
		PluginConfigJSON:      r.plugin.PluginConfigJSON,
	}

	if r.nextConsumerM != nil {
		r.wg.Add(1)
		go r.runMetrics(ctx)
	}

	if r.nextConsumerL != nil {
		r.wg.Add(1)
		go r.runLogs(ctx)
	}

	if r.nextConsumerT != nil {
		r.wg.Add(1)
		go r.runTraces(ctx)
	}

	return nil
}

func (r *Receiver) runMetrics(ctx context.Context) error {
	fmt.Println("runMetrics called")
	defer r.wg.Done()

	_, err := r.plugin.ProcessFunctionCall(ctx, "startMetricsReceiver", r.stack)
	if err != nil {
		return err
	}

	return nil
}

func (r *Receiver) runLogs(ctx context.Context) error {
	fmt.Println("runLogs called")
	defer r.wg.Done()

	_, err := r.plugin.ProcessFunctionCall(ctx, "startLogsReceiver", r.stack)
	if err != nil {
		fmt.Println("Error in runLogs:", err)
		return err
	}

	fmt.Println("runLogs completed")

	return nil
}

func (r *Receiver) runTraces(ctx context.Context) error {
	fmt.Println("runTraces called")
	defer r.wg.Done()

	_, err := r.plugin.ProcessFunctionCall(ctx, "startTracesReceiver", r.stack)
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
