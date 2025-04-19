package api

import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type Plugin interface{}

// LogsReceiver is the interface for implementing a logs receiver
type LogsReceiver interface {
	Plugin

	// StartLogs with context for long-running receivers
	StartLogs(ctx context.Context)
}

type MetricsReceiver interface {
	Plugin

	StartMetrics(ctx context.Context)
}

type TracesProcessor interface {
	Plugin

	ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *Status)
}

type MetricsProcessor interface {
	Plugin

	ProcessMetrics(metrics pmetric.Metrics) (pmetric.Metrics, *Status)
}

type LogsProcessor interface {
	Plugin

	ProcessLogs(logs plog.Logs) (plog.Logs, *Status)
}

type TracesExporter interface {
	Plugin

	PushTraces(traces ptrace.Traces) *Status
}

type MetricsExporter interface {
	Plugin

	PushMetrics(metrics pmetric.Metrics) *Status
}

type LogsExporter interface {
	Plugin

	PushLogs(logs plog.Logs) *Status
}
