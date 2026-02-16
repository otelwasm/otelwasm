package wasmreceiver

import (
	"context"
	"fmt"
	"sync"

	"github.com/otelwasm/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

const (
	startMetricsReceiverFunctionName = "otelwasm_start_metrics_receiver"
	startLogsReceiverFunctionName    = "otelwasm_start_logs_receiver"
	startTracesReceiverFunctionName  = "otelwasm_start_traces_receiver"
)

type Receiver struct {
	cfg           *Config
	set           receiver.Settings
	plugin        *wasmplugin.WasmPlugin
	nextConsumerM consumer.Metrics
	nextConsumerL consumer.Logs
	nextConsumerT consumer.Traces

	stack *wasmplugin.Stack
	wg    sync.WaitGroup
}

func newMetricsWasmReceiver(ctx context.Context, cfg *Config, nextConsumerM consumer.Metrics, set receiver.Settings) (context.Context, *Receiver, error) {
	if err := cfg.Validate(); err != nil {
		return ctx, nil, err
	}

	requiredFunctions := []string{startMetricsReceiverFunctionName}

	plugin, err := wasmplugin.NewWasmPlugin(ctx, &cfg.Config, requiredFunctions)
	if err != nil {
		return ctx, nil, err
	}

	if supported, err := plugin.IsMetricsSupported(ctx); err != nil {
		return ctx, nil, fmt.Errorf("failed to check metrics support status: %w", err)
	} else if !supported {
		return ctx, nil, pipeline.ErrSignalNotSupported
	}

	return ctx, &Receiver{
		cfg:           cfg,
		plugin:        plugin,
		nextConsumerM: nextConsumerM,
		set:           set,
	}, nil
}

func newLogsWasmReceiver(ctx context.Context, cfg *Config, nextConsumerL consumer.Logs, set receiver.Settings) (context.Context, *Receiver, error) {
	if err := cfg.Validate(); err != nil {
		return ctx, nil, err
	}

	requiredFunctions := []string{startLogsReceiverFunctionName}

	plugin, err := wasmplugin.NewWasmPlugin(ctx, &cfg.Config, requiredFunctions)
	if err != nil {
		return ctx, nil, err
	}

	if supported, err := plugin.IsLogsSupported(ctx); err != nil {
		return ctx, nil, fmt.Errorf("failed to check logs support status: %w", err)
	} else if !supported {
		return ctx, nil, pipeline.ErrSignalNotSupported
	}

	return ctx, &Receiver{
		cfg:           cfg,
		plugin:        plugin,
		nextConsumerL: nextConsumerL,
		set:           set,
	}, nil
}

func newTracesWasmReceiver(ctx context.Context, cfg *Config, nextConsumerT consumer.Traces, set receiver.Settings) (context.Context, *Receiver, error) {
	if err := cfg.Validate(); err != nil {
		return ctx, nil, err
	}

	requiredFunctions := []string{startTracesReceiverFunctionName}

	plugin, err := wasmplugin.NewWasmPlugin(ctx, &cfg.Config, requiredFunctions)
	if err != nil {
		return ctx, nil, err
	}

	if supported, err := plugin.IsTracesSupported(ctx); err != nil {
		return ctx, nil, fmt.Errorf("failed to check traces support status: %w", err)
	} else if !supported {
		return ctx, nil, pipeline.ErrSignalNotSupported
	}

	return ctx, &Receiver{
		cfg:           cfg,
		plugin:        plugin,
		nextConsumerT: nextConsumerT,
		set:           set,
	}, nil
}

var (
	_ receiver.Metrics = (*Receiver)(nil)
	_ receiver.Logs    = (*Receiver)(nil)
	_ receiver.Traces  = (*Receiver)(nil)
)

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

func (r *Receiver) runMetrics(ctx context.Context) {
	defer r.wg.Done()

	_, err := r.plugin.ProcessFunctionCall(ctx, startMetricsReceiverFunctionName, r.stack)
	if err != nil {
		r.set.Logger.Fatal("metrics receiver failed", zap.Error(err))
	}
}

func (r *Receiver) runLogs(ctx context.Context) {
	defer r.wg.Done()

	_, err := r.plugin.ProcessFunctionCall(ctx, startLogsReceiverFunctionName, r.stack)
	if err != nil {
		r.set.Logger.Fatal("logs receiver failed", zap.Error(err))
	}
}

func (r *Receiver) runTraces(ctx context.Context) {
	defer r.wg.Done()

	_, err := r.plugin.ProcessFunctionCall(ctx, startTracesReceiverFunctionName, r.stack)
	if err != nil {
		r.set.Logger.Fatal("traces receiver failed", zap.Error(err))
	}
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
	if r.stack != nil {
		r.stack.RequestedShutdown.Store(true)
	}
	// TODO: Set timeout for shutdown

	r.wg.Wait()

	return r.plugin.Shutdown(ctx)
}
