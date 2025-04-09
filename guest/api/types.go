package api

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type Plugin interface{}

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
