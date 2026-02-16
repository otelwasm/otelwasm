package plugin

import (
	"context"
	"runtime"
	"time"

	"github.com/otelwasm/otelwasm/guest/api"
	pubimports "github.com/otelwasm/otelwasm/guest/imports"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/mem"
	intplugin "github.com/otelwasm/otelwasm/guest/internal/plugin"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	tracesProcessor  api.TracesProcessor
	metricsProcessor api.MetricsProcessor
	logsProcessor    api.LogsProcessor
	tracesExporter   api.TracesExporter
	metricsExporter  api.MetricsExporter
	logsExporter     api.LogsExporter
	tracesReceiver   api.TracesReceiver
	metricsReceiver  api.MetricsReceiver
	logsReceiver     api.LogsReceiver
)

func Set(p api.Plugin) {
	if p == nil {
		panic("nil plugin")
	}
	intplugin.MustSet(p)

	if tp, ok := p.(api.TracesProcessor); ok {
		tracesProcessor = tp
		supportedTelemetry |= telemetryTypeTraces
	}
	if mp, ok := p.(api.MetricsProcessor); ok {
		metricsProcessor = mp
		supportedTelemetry |= telemetryTypeMetrics
	}
	if lp, ok := p.(api.LogsProcessor); ok {
		logsProcessor = lp
		supportedTelemetry |= telemetryTypeLogs
	}
	if te, ok := p.(api.TracesExporter); ok {
		tracesExporter = te
		supportedTelemetry |= telemetryTypeTraces
	}
	if me, ok := p.(api.MetricsExporter); ok {
		metricsExporter = me
		supportedTelemetry |= telemetryTypeMetrics
	}
	if le, ok := p.(api.LogsExporter); ok {
		logsExporter = le
		supportedTelemetry |= telemetryTypeLogs
	}
	if tr, ok := p.(api.TracesReceiver); ok {
		tracesReceiver = tr
		supportedTelemetry |= telemetryTypeTraces
	}
	if mr, ok := p.(api.MetricsReceiver); ok {
		metricsReceiver = mr
		supportedTelemetry |= telemetryTypeMetrics
	}
	if lr, ok := p.(api.LogsReceiver); ok {
		logsReceiver = lr
		supportedTelemetry |= telemetryTypeLogs
	}
}

//go:wasmexport otelwasm_consume_traces
func _consumeTraces(dataPtr uint32, dataSize uint32) uint32 {
	raw := mem.TakeOwnership(dataPtr, dataSize)
	unmarshaler := ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(raw)
	if err != nil {
		return imports.StatusToCode(api.StatusError(err.Error()))
	}

	if tracesProcessor != nil {
		result, status := tracesProcessor.ProcessTraces(traces)
		if result != (ptrace.Traces{}) {
			pubimports.SetResultTraces(result)
		}
		runtime.KeepAlive(result)
		return imports.StatusToCode(status)
	}
	if tracesExporter != nil {
		return imports.StatusToCode(tracesExporter.PushTraces(traces))
	}
	return imports.StatusToCode(api.StatusError("traces telemetry is not supported"))
}

//go:wasmexport otelwasm_consume_metrics
func _consumeMetrics(dataPtr uint32, dataSize uint32) uint32 {
	raw := mem.TakeOwnership(dataPtr, dataSize)
	unmarshaler := pmetric.ProtoUnmarshaler{}
	metrics, err := unmarshaler.UnmarshalMetrics(raw)
	if err != nil {
		return imports.StatusToCode(api.StatusError(err.Error()))
	}

	if metricsProcessor != nil {
		result, status := metricsProcessor.ProcessMetrics(metrics)
		if result != (pmetric.Metrics{}) {
			pubimports.SetResultMetrics(result)
		}
		runtime.KeepAlive(result)
		return imports.StatusToCode(status)
	}
	if metricsExporter != nil {
		return imports.StatusToCode(metricsExporter.PushMetrics(metrics))
	}
	return imports.StatusToCode(api.StatusError("metrics telemetry is not supported"))
}

//go:wasmexport otelwasm_consume_logs
func _consumeLogs(dataPtr uint32, dataSize uint32) uint32 {
	raw := mem.TakeOwnership(dataPtr, dataSize)
	unmarshaler := plog.ProtoUnmarshaler{}
	logs, err := unmarshaler.UnmarshalLogs(raw)
	if err != nil {
		return imports.StatusToCode(api.StatusError(err.Error()))
	}

	if logsProcessor != nil {
		result, status := logsProcessor.ProcessLogs(logs)
		if result != (plog.Logs{}) {
			pubimports.SetResultLogs(result)
		}
		runtime.KeepAlive(result)
		return imports.StatusToCode(status)
	}
	if logsExporter != nil {
		return imports.StatusToCode(logsExporter.PushLogs(logs))
	}
	return imports.StatusToCode(api.StatusError("logs telemetry is not supported"))
}

//go:wasmexport otelwasm_start_traces_receiver
func _startTracesReceiver() {
	runReceiverLoop(func(ctx context.Context) {
		if tracesReceiver != nil {
			tracesReceiver.StartTraces(ctx)
		}
	})
}

//go:wasmexport otelwasm_start_metrics_receiver
func _startMetricsReceiver() {
	runReceiverLoop(func(ctx context.Context) {
		if metricsReceiver != nil {
			metricsReceiver.StartMetrics(ctx)
		}
	})
}

//go:wasmexport otelwasm_start_logs_receiver
func _startLogsReceiver() {
	runReceiverLoop(func(ctx context.Context) {
		if logsReceiver != nil {
			logsReceiver.StartLogs(ctx)
		}
	})
}

func runReceiverLoop(startFunc func(context.Context)) {
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

	startFunc(ctx)
}
