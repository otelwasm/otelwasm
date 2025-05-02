package imports

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/otelwasm/otelwasm/guest/internal/mem"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func GetConfig(v any) error {
	rawMsg := mem.GetBytes(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return getPluginConfig(ptr, limit)
	})
	return json.Unmarshal(rawMsg, v)
}

func SetResultTraces(traces ptrace.Traces) {
	marshaler := ptrace.ProtoMarshaler{}
	rawMsg, err := marshaler.MarshalTraces(traces)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	ptr, size := mem.BytesToPtr(rawMsg)
	setResultTraces(ptr, size)
	runtime.KeepAlive(rawMsg) // until ptr is no longer needed
}

func SetResultMetrics(metrics pmetric.Metrics) {
	marshaler := pmetric.ProtoMarshaler{}
	rawMsg, err := marshaler.MarshalMetrics(metrics)
	if err != nil {
		panic(err)
	}
	ptr, size := mem.BytesToPtr(rawMsg)
	setResultMetrics(ptr, size)
	runtime.KeepAlive(rawMsg) // until ptr is no longer needed
}

func SetResultLogs(logs plog.Logs) {
	marshaler := plog.ProtoMarshaler{}
	rawMsg, err := marshaler.MarshalLogs(logs)
	if err != nil {
		panic(err)
	}
	ptr, size := mem.BytesToPtr(rawMsg)
	setResultLogs(ptr, size)
	runtime.KeepAlive(rawMsg) // until ptr is no longer needed
}
