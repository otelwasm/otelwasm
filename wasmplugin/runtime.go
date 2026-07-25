package wasmplugin

import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Memory provides access to the guest module's linear memory.
// It is implemented by each WebAssembly runtime backend.
type Memory interface {
	// Read reads byteCount bytes from the underlying buffer at the offset,
	// or returns false if out of range.
	Read(offset, byteCount uint32) ([]byte, bool)

	// Write writes the bytes to the underlying buffer at the offset,
	// or returns false if out of range.
	Write(offset uint32, v []byte) bool
}

// guestRuntime abstracts a WebAssembly runtime hosting the guest module.
// Implementations exist for wazero (pure Go, always available) and
// WasmEdge (CGO, enabled with the "wasmedge" build tag).
type guestRuntime interface {
	// CallFunction invokes an exported guest function.
	// The stack is made accessible to the host functions
	// (opentelemetry.io/wasm module) for the duration of the call.
	CallFunction(ctx context.Context, name string, stack *Stack) ([]uint64, error)

	// HasFunction reports whether the guest module exports a function with
	// the given name.
	HasFunction(name string) bool

	// Close releases all resources held by the runtime.
	Close(ctx context.Context) error
}

// Host function implementations shared by all runtime backends.
// Each backend adapts these to its own host function calling convention.

func hostCurrentTraces(stack *Stack, mem Memory, buf, bufLimit uint32) uint32 {
	return marshalTraceIfUnderLimit(mem, stack.CurrentTraces, buf, bufLimit)
}

func hostCurrentMetrics(stack *Stack, mem Memory, buf, bufLimit uint32) uint32 {
	return marshalMetricsIfUnderLimit(mem, stack.CurrentMetrics, buf, bufLimit)
}

func hostCurrentLogs(stack *Stack, mem Memory, buf, bufLimit uint32) uint32 {
	return marshalLogsIfUnderLimit(mem, stack.CurrentLogs, buf, bufLimit)
}

func hostGetPluginConfig(stack *Stack, mem Memory, buf, bufLimit uint32) uint32 {
	return writeBytesIfUnderLimit(mem, stack.PluginConfigJSON, buf, bufLimit)
}

func hostGetShutdownRequested(stack *Stack) uint32 {
	if stack.RequestedShutdown.Load() {
		return 1
	}
	return 0
}

func hostSetResultTraces(stack *Stack, mem Memory, buf, size uint32) {
	// Read the serialized traces from WASM memory
	tracesBytes, ok := mem.Read(buf, size)
	if !ok {
		panic("out of memory reading result traces") // Bug: caller passed a length outside memory
	}

	// Unmarshal the traces
	unmarshaler := ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(tracesBytes)
	if err != nil {
		panic(err) // Bug: in unmarshaller
	}

	// Store the result traces in the stack
	stack.ResultTraces = traces
	if stack.OnResultTracesChange != nil {
		stack.OnResultTracesChange(traces)
	}
}

func hostSetResultMetrics(stack *Stack, mem Memory, buf, size uint32) {
	// Read the serialized metrics from WASM memory
	metricsBytes, ok := mem.Read(buf, size)
	if !ok {
		panic("out of memory reading result metrics") // Bug: caller passed a length outside memory
	}

	// Unmarshal the metrics
	unmarshaler := pmetric.ProtoUnmarshaler{}
	metrics, err := unmarshaler.UnmarshalMetrics(metricsBytes)
	if err != nil {
		panic(err) // Bug: in unmarshaller
	}

	// Store the result metrics in the stack
	stack.ResultMetrics = metrics
	if stack.OnResultMetricsChange != nil {
		stack.OnResultMetricsChange(metrics)
	}
}

func hostSetResultLogs(stack *Stack, mem Memory, buf, size uint32) {
	// Read the serialized logs from WASM memory
	logsBytes, ok := mem.Read(buf, size)
	if !ok {
		panic("out of memory reading result logs") // Bug: caller passed a length outside memory
	}

	// Unmarshal the logs
	unmarshaler := plog.ProtoUnmarshaler{}
	logs, err := unmarshaler.UnmarshalLogs(logsBytes)
	if err != nil {
		panic(err) // Bug: in unmarshaller
	}

	// Store the result logs in the stack
	stack.ResultLogs = logs
	if stack.OnResultLogsChange != nil {
		stack.OnResultLogsChange(logs)
	}
}

func hostSetResultStatusReason(stack *Stack, mem Memory, buf, size uint32) {
	// Read the status reason string from WASM memory
	reasonBytes, ok := mem.Read(buf, size)
	if !ok {
		panic("out of memory reading status reason") // Bug: caller passed a length outside memory
	}

	// Store the status reason in the stack
	stack.StatusReason = string(reasonBytes)
}
