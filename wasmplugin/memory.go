package wasmplugin

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// These utility functions are derived from the kube-scheduler-wasm-extension.
// https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension

// writeBytesIfUnderLimit writes bytes to memory if they fit within the limit.
// It returns the size of the bytes regardless, so that the guest can grow its
// buffer and retry when the value didn't fit (see guest/internal/mem.GetBytes).
func writeBytesIfUnderLimit(memory Memory, bytes []byte, buf, bufLimit uint32) uint32 {
	size := uint32(len(bytes))
	if size == 0 || size > bufLimit {
		return size
	}
	if !memory.Write(buf, bytes) {
		return 0
	}
	return size
}

// marshalTraceIfUnderLimit marshals traces to memory if they fit within the limit
func marshalTraceIfUnderLimit(memory Memory, traces ptrace.Traces, buf, bufLimit uint32) uint32 {
	marshaler := ptrace.ProtoMarshaler{}
	tracesBytes, err := marshaler.MarshalTraces(traces)
	if err != nil {
		return 0
	}
	return writeBytesIfUnderLimit(memory, tracesBytes, buf, bufLimit)
}

// marshalMetricsIfUnderLimit marshals metrics to memory if they fit within the limit
func marshalMetricsIfUnderLimit(memory Memory, metrics pmetric.Metrics, buf, bufLimit uint32) uint32 {
	marshaler := pmetric.ProtoMarshaler{}
	metricsBytes, err := marshaler.MarshalMetrics(metrics)
	if err != nil {
		return 0
	}
	return writeBytesIfUnderLimit(memory, metricsBytes, buf, bufLimit)
}

// marshalLogsIfUnderLimit marshals logs to memory if they fit within the limit
func marshalLogsIfUnderLimit(memory Memory, logs plog.Logs, buf, bufLimit uint32) uint32 {
	marshaler := plog.ProtoMarshaler{}
	logsBytes, err := marshaler.MarshalLogs(logs)
	if err != nil {
		return 0
	}
	return writeBytesIfUnderLimit(memory, logsBytes, buf, bufLimit)
}
