package wasmplugin

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero/api"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// This file implements a handle/accessor based ABI for pdata, which allows
// guests to read and mutate telemetry held by the host without serializing
// the whole payload across the host/guest boundary.
//
// Objects are referenced by opaque handles: uint32 indices into a per-call
// table stored in the Stack. Handles are only valid for the duration of a
// single guest function call.
//
// Field IDs are the OTLP protobuf field numbers of the owning message, so
// both sides can derive constants from the same .proto definitions. Since
// field numbers are scoped to a message, dispatch is on the handle's dynamic
// type first, then the field ID.

// Host function exports for the handle-based ABI
const (
	pdataCurrentTraces       = "pdataCurrentTraces"
	pdataCommitCurrentTraces = "pdataCommitCurrentTraces"
	pdataGetI64              = "pdataGetI64"
	pdataSetI64              = "pdataSetI64"
	pdataGetStr              = "pdataGetStr"
	pdataSetStr              = "pdataSetStr"
	pdataGetMsg              = "pdataGetMsg"
	pdataSliceLen            = "pdataSliceLen"
	pdataSliceGet            = "pdataSliceGet"
	pdataMapLen              = "pdataMapLen"
	pdataMapGetStr           = "pdataMapGetStr"
	pdataMapPutStr           = "pdataMapPutStr"
)

// Field IDs are OTLP protobuf field numbers.
// https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto
const (
	fieldTracesResourceSpans = 1

	fieldResourceSpansResource   = 1
	fieldResourceSpansScopeSpans = 2
	fieldResourceSpansSchemaUrl  = 3

	fieldScopeSpansScope     = 1
	fieldScopeSpansSpans     = 2
	fieldScopeSpansSchemaUrl = 3

	fieldSpanTraceState     = 3
	fieldSpanName           = 5
	fieldSpanKind           = 6
	fieldSpanStartTime      = 7
	fieldSpanEndTime        = 8
	fieldSpanAttributes     = 9
	fieldSpanDroppedAttribs = 10
	fieldSpanStatus         = 15
	fieldSpanFlags          = 16

	fieldStatusMessage = 2
	fieldStatusCode    = 3

	// https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/resource/v1/resource.proto
	fieldResourceAttributes     = 1
	fieldResourceDroppedAttribs = 2

	// https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/common/v1/common.proto
	fieldScopeName           = 1
	fieldScopeVersion        = 2
	fieldScopeAttributes     = 3
	fieldScopeDroppedAttribs = 4
)

// addHandle registers obj in the per-call handle table and returns its handle.
// Handle 0 is reserved as invalid.
func (s *Stack) addHandle(obj any) uint32 {
	s.handles = append(s.handles, obj)
	return uint32(len(s.handles))
}

// getHandle resolves a handle previously returned by addHandle.
func (s *Stack) getHandle(id uint32) any {
	if id == 0 || int(id) > len(s.handles) {
		panic(fmt.Sprintf("invalid pdata handle: %d", id)) // Bug: guest passed an unknown handle
	}
	return s.handles[id-1]
}

// writeStrResult writes s into guest memory if it fits in bufLimit and
// returns the byte length of s regardless, so the guest can retry with a
// larger buffer.
func writeStrResult(memory api.Memory, s string, buf, bufLimit uint32) uint32 {
	size := uint32(len(s))
	if size == 0 || size > bufLimit {
		return size
	}
	if !memory.WriteString(buf, s) {
		panic("out of memory writing string result") // Bug: guest passed a buffer outside memory
	}
	return size
}

// readGuestString reads a string passed from the guest by pointer and length.
func readGuestString(memory api.Memory, ptr, size uint32) string {
	bytes, ok := memory.Read(ptr, size)
	if !ok {
		panic("out of memory reading guest string") // Bug: guest passed a pointer outside memory
	}
	return string(bytes)
}

func pdataCurrentTracesFn(ctx context.Context, mod api.Module, stack []uint64) {
	s := paramsFromContext(ctx)
	stack[0] = uint64(s.addHandle(s.CurrentTraces))
}

func pdataCommitCurrentTracesFn(ctx context.Context, mod api.Module, stack []uint64) {
	s := paramsFromContext(ctx)
	s.ResultTraces = s.CurrentTraces
	if s.OnResultTracesChange != nil {
		s.OnResultTracesChange(s.ResultTraces)
	}
}

func pdataGetI64Fn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	field := uint32(stack[1])

	var v int64
	switch o := paramsFromContext(ctx).getHandle(h).(type) {
	case ptrace.Span:
		switch field {
		case fieldSpanKind:
			v = int64(o.Kind())
		case fieldSpanStartTime:
			v = int64(o.StartTimestamp())
		case fieldSpanEndTime:
			v = int64(o.EndTimestamp())
		case fieldSpanDroppedAttribs:
			v = int64(o.DroppedAttributesCount())
		case fieldSpanFlags:
			v = int64(o.Flags())
		default:
			panicUnknownField("Span", field)
		}
	case ptrace.Status:
		switch field {
		case fieldStatusCode:
			v = int64(o.Code())
		default:
			panicUnknownField("Status", field)
		}
	case pcommon.Resource:
		switch field {
		case fieldResourceDroppedAttribs:
			v = int64(o.DroppedAttributesCount())
		default:
			panicUnknownField("Resource", field)
		}
	case pcommon.InstrumentationScope:
		switch field {
		case fieldScopeDroppedAttribs:
			v = int64(o.DroppedAttributesCount())
		default:
			panicUnknownField("InstrumentationScope", field)
		}
	default:
		panicUnknownType("pdataGetI64", o)
	}
	stack[0] = uint64(v)
}

func pdataSetI64Fn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	field := uint32(stack[1])
	v := int64(stack[2])

	switch o := paramsFromContext(ctx).getHandle(h).(type) {
	case ptrace.Span:
		switch field {
		case fieldSpanKind:
			o.SetKind(ptrace.SpanKind(v))
		case fieldSpanStartTime:
			o.SetStartTimestamp(pcommon.Timestamp(v))
		case fieldSpanEndTime:
			o.SetEndTimestamp(pcommon.Timestamp(v))
		case fieldSpanDroppedAttribs:
			o.SetDroppedAttributesCount(uint32(v))
		case fieldSpanFlags:
			o.SetFlags(uint32(v))
		default:
			panicUnknownField("Span", field)
		}
	case ptrace.Status:
		switch field {
		case fieldStatusCode:
			o.SetCode(ptrace.StatusCode(v))
		default:
			panicUnknownField("Status", field)
		}
	default:
		panicUnknownType("pdataSetI64", o)
	}
}

func pdataGetStrFn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	field := uint32(stack[1])
	buf := uint32(stack[2])
	bufLimit := uint32(stack[3])

	var v string
	switch o := paramsFromContext(ctx).getHandle(h).(type) {
	case ptrace.ResourceSpans:
		switch field {
		case fieldResourceSpansSchemaUrl:
			v = o.SchemaUrl()
		default:
			panicUnknownField("ResourceSpans", field)
		}
	case ptrace.ScopeSpans:
		switch field {
		case fieldScopeSpansSchemaUrl:
			v = o.SchemaUrl()
		default:
			panicUnknownField("ScopeSpans", field)
		}
	case ptrace.Span:
		switch field {
		case fieldSpanName:
			v = o.Name()
		case fieldSpanTraceState:
			v = o.TraceState().AsRaw()
		default:
			panicUnknownField("Span", field)
		}
	case ptrace.Status:
		switch field {
		case fieldStatusMessage:
			v = o.Message()
		default:
			panicUnknownField("Status", field)
		}
	case pcommon.InstrumentationScope:
		switch field {
		case fieldScopeName:
			v = o.Name()
		case fieldScopeVersion:
			v = o.Version()
		default:
			panicUnknownField("InstrumentationScope", field)
		}
	default:
		panicUnknownType("pdataGetStr", o)
	}
	stack[0] = uint64(writeStrResult(mod.Memory(), v, buf, bufLimit))
}

func pdataSetStrFn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	field := uint32(stack[1])
	v := readGuestString(mod.Memory(), uint32(stack[2]), uint32(stack[3]))

	switch o := paramsFromContext(ctx).getHandle(h).(type) {
	case ptrace.ResourceSpans:
		switch field {
		case fieldResourceSpansSchemaUrl:
			o.SetSchemaUrl(v)
		default:
			panicUnknownField("ResourceSpans", field)
		}
	case ptrace.ScopeSpans:
		switch field {
		case fieldScopeSpansSchemaUrl:
			o.SetSchemaUrl(v)
		default:
			panicUnknownField("ScopeSpans", field)
		}
	case ptrace.Span:
		switch field {
		case fieldSpanName:
			o.SetName(v)
		case fieldSpanTraceState:
			o.TraceState().FromRaw(v)
		default:
			panicUnknownField("Span", field)
		}
	case ptrace.Status:
		switch field {
		case fieldStatusMessage:
			o.SetMessage(v)
		default:
			panicUnknownField("Status", field)
		}
	case pcommon.InstrumentationScope:
		switch field {
		case fieldScopeName:
			o.SetName(v)
		case fieldScopeVersion:
			o.SetVersion(v)
		default:
			panicUnknownField("InstrumentationScope", field)
		}
	default:
		panicUnknownType("pdataSetStr", o)
	}
}

func pdataGetMsgFn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	field := uint32(stack[1])

	s := paramsFromContext(ctx)
	var child any
	switch o := s.getHandle(h).(type) {
	case ptrace.Traces:
		switch field {
		case fieldTracesResourceSpans:
			child = o.ResourceSpans()
		default:
			panicUnknownField("Traces", field)
		}
	case ptrace.ResourceSpans:
		switch field {
		case fieldResourceSpansResource:
			child = o.Resource()
		case fieldResourceSpansScopeSpans:
			child = o.ScopeSpans()
		default:
			panicUnknownField("ResourceSpans", field)
		}
	case ptrace.ScopeSpans:
		switch field {
		case fieldScopeSpansScope:
			child = o.Scope()
		case fieldScopeSpansSpans:
			child = o.Spans()
		default:
			panicUnknownField("ScopeSpans", field)
		}
	case ptrace.Span:
		switch field {
		case fieldSpanAttributes:
			child = o.Attributes()
		case fieldSpanStatus:
			child = o.Status()
		default:
			panicUnknownField("Span", field)
		}
	case pcommon.Resource:
		switch field {
		case fieldResourceAttributes:
			child = o.Attributes()
		default:
			panicUnknownField("Resource", field)
		}
	case pcommon.InstrumentationScope:
		switch field {
		case fieldScopeAttributes:
			child = o.Attributes()
		default:
			panicUnknownField("InstrumentationScope", field)
		}
	default:
		panicUnknownType("pdataGetMsg", o)
	}
	stack[0] = uint64(s.addHandle(child))
}

func pdataSliceLenFn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])

	var n int
	switch o := paramsFromContext(ctx).getHandle(h).(type) {
	case ptrace.ResourceSpansSlice:
		n = o.Len()
	case ptrace.ScopeSpansSlice:
		n = o.Len()
	case ptrace.SpanSlice:
		n = o.Len()
	default:
		panicUnknownType("pdataSliceLen", o)
	}
	stack[0] = uint64(uint32(n))
}

func pdataSliceGetFn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	index := int(uint32(stack[1]))

	s := paramsFromContext(ctx)
	var elem any
	switch o := s.getHandle(h).(type) {
	case ptrace.ResourceSpansSlice:
		elem = o.At(index)
	case ptrace.ScopeSpansSlice:
		elem = o.At(index)
	case ptrace.SpanSlice:
		elem = o.At(index)
	default:
		panicUnknownType("pdataSliceGet", o)
	}
	stack[0] = uint64(s.addHandle(elem))
}

func pdataMapLenFn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	m := paramsFromContext(ctx).getHandle(h).(pcommon.Map)
	stack[0] = uint64(uint32(m.Len()))
}

func pdataMapGetStrFn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	key := readGuestString(mod.Memory(), uint32(stack[1]), uint32(stack[2]))
	buf := uint32(stack[3])
	bufLimit := uint32(stack[4])

	m := paramsFromContext(ctx).getHandle(h).(pcommon.Map)
	v, ok := m.Get(key)
	if !ok {
		stack[0] = 0
		return
	}
	stack[0] = uint64(writeStrResult(mod.Memory(), v.AsString(), buf, bufLimit))
}

func pdataMapPutStrFn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	key := readGuestString(mod.Memory(), uint32(stack[1]), uint32(stack[2]))
	val := readGuestString(mod.Memory(), uint32(stack[3]), uint32(stack[4]))

	m := paramsFromContext(ctx).getHandle(h).(pcommon.Map)
	m.PutStr(key, val)
}

func panicUnknownField(typeName string, field uint32) {
	panic(fmt.Sprintf("unknown field %d for %s", field, typeName)) // Bug: guest passed an unsupported field ID
}

func panicUnknownType(funcName string, obj any) {
	panic(fmt.Sprintf("%s: unsupported handle type %T", funcName, obj)) // Bug: guest passed a handle of the wrong type
}
