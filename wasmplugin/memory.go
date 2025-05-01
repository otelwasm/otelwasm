package wasmplugin

import (
	"github.com/tetratelabs/wazero/api"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// These utility functions are derived from the kube-scheduler-wasm-extension.
// https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension

// writeBytesIfUnderLimit writes bytes to memory if they fit within the limit
func writeBytesIfUnderLimit(memory api.Memory, bytes []byte, buf, bufLimit uint32) uint32 {
	if uint32(len(bytes)) > bufLimit {
		return 0
	}
	if !memory.Write(buf, bytes) {
		return 0
	}
	return uint32(len(bytes))
}

// marshalTraceIfUnderLimit marshals traces to memory if they fit within the limit
func marshalTraceIfUnderLimit(memory api.Memory, traces ptrace.Traces, buf, bufLimit uint32) uint32 {
	marshaler := ptrace.ProtoMarshaler{}
	tracesBytes, err := marshaler.MarshalTraces(traces)
	if err != nil {
		return 0
	}
	return writeBytesIfUnderLimit(memory, tracesBytes, buf, bufLimit)
}

// marshalMetricsIfUnderLimit marshals metrics to memory if they fit within the limit
func marshalMetricsIfUnderLimit(memory api.Memory, metrics pmetric.Metrics, buf, bufLimit uint32) uint32 {
	marshaler := pmetric.ProtoMarshaler{}
	metricsBytes, err := marshaler.MarshalMetrics(metrics)
	if err != nil {
		return 0
	}
	return writeBytesIfUnderLimit(memory, metricsBytes, buf, bufLimit)
}

// marshalLogsIfUnderLimit marshals logs to memory if they fit within the limit
func marshalLogsIfUnderLimit(memory api.Memory, logs plog.Logs, buf, bufLimit uint32) uint32 {
	marshaler := plog.ProtoMarshaler{}
	logsBytes, err := marshaler.MarshalLogs(logs)
	if err != nil {
		return 0
	}
	return writeBytesIfUnderLimit(memory, logsBytes, buf, bufLimit)
}
