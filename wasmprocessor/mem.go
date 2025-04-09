package wasmprocessor

import (
	wazeroapi "github.com/tetratelabs/wazero/api"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// bufLimit is the possibly zero maximum length of a result value to write in
// bytes. If the actual value is larger than this, nothing is written to
// memory.
type bufLimit = uint32

func marshalTraceIfUnderLimit(mem wazeroapi.Memory, dt ptrace.Traces, buf uint32, bufLimit bufLimit) int {
	marshaler := ptrace.ProtoMarshaler{}
	vLen := marshaler.TracesSize(dt)
	if vLen == 0 {
		return 0 // nothing to write
	}

	// Next, see if the value will fit inside the buffer.
	if vLen > int(bufLimit) {
		// If it doesn't fit, the caller can decide to retry with a larger
		// buffer or fail.
		return vLen
	}

	// Now, we know the value isn't too large to fit in the buffer. Write it
	// directly to the Wasm memory.
	wasmMem, ok := mem.Read(buf, uint32(vLen))
	if !ok {
		panic("out of memory") // Bug: caller passed a length outside memory
	}

	b, err := marshaler.MarshalTraces(dt)
	if err != nil {
		panic(err) // Bug: in marshaller.
	}

	copy(wasmMem, b)

	// Success: return the bytes written, so that the caller can unmarshal from
	// a sized buffer.
	return vLen
}

func marshalMetricsIfUnderLimit(mem wazeroapi.Memory, dt pmetric.Metrics, buf uint32, bufLimit bufLimit) int {
	marshaler := pmetric.ProtoMarshaler{}
	vLen := marshaler.MetricsSize(dt)
	if vLen == 0 {
		return 0 // nothing to write
	}

	// Next, see if the value will fit inside the buffer.
	if vLen > int(bufLimit) {
		// If it doesn't fit, the caller can decide to retry with a larger
		// buffer or fail.
		return vLen
	}

	// Now, we know the value isn't too large to fit in the buffer. Write it
	// directly to the Wasm memory.
	wasmMem, ok := mem.Read(buf, uint32(vLen))
	if !ok {
		panic("out of memory") // Bug: caller passed a length outside memory
	}

	b, err := marshaler.MarshalMetrics(dt)
	if err != nil {
		panic(err) // Bug: in marshaller.
	}

	copy(wasmMem, b)

	// Success: return the bytes written, so that the caller can unmarshal from
	// a sized buffer.
	return vLen
}
