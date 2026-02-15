package imports

import (
	"runtime"

	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/internal/mem"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// StatusToCode returns a WebAssembly compatible result for the input status,
// after sending any reason to the host.
func StatusToCode(s *api.Status) uint32 {
	// Nil status is the same as one with a success code.
	if s == nil || s.Code == api.StatusCodeSuccess {
		return uint32(api.StatusCodeSuccess)
	}

	// WebAssembly Core 2.0 (DRAFT) only includes numeric types. Return the
	// reason using a host function.
	if reason := s.Reason; reason != "" {
		setStatusReason(reason)
	}

	return uint32(s.Code)
}

func setStatusReason(reason string) {
	ptr, size := mem.StringToPtr(reason)
	setStatusReasonHost(ptr, size)
	runtime.KeepAlive(reason) // until ptr is no longer needed.
}

func CurrentTraces() ptrace.Traces {
	rawMsg := mem.GetBytes(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return currentTraces(ptr, limit)
	})
	unmarshaler := ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(rawMsg)
	if err != nil {
		panic(err)
	}
	return traces
}

func CurrentMetrics() pmetric.Metrics {
	rawMsg := mem.GetBytes(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return currentMetrics(ptr, limit)
	})
	unmarshaler := pmetric.ProtoUnmarshaler{}
	metrics, err := unmarshaler.UnmarshalMetrics(rawMsg)
	if err != nil {
		panic(err)
	}
	return metrics
}

func CurrentLogs() plog.Logs {
	rawMsg := mem.GetBytes(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return currentLogs(ptr, limit)
	})
	unmarshaler := plog.ProtoUnmarshaler{}
	logs, err := unmarshaler.UnmarshalLogs(rawMsg)
	if err != nil {
		panic(err)
	}
	return logs
}

func GetShutdownRequested() bool {
	return getShutdownRequested() != 0
}
