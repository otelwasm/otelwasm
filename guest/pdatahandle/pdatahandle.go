// Package pdatahandle provides a handle/accessor based view of the telemetry
// held by the host. Instead of copying the whole payload into guest memory
// and unmarshaling it (see guest/imports.CurrentTraces), telemetry stays on
// the host and is read or mutated through fine-grained host functions.
//
// Handles are only valid for the duration of a single exported function call;
// fetch a fresh root with CurrentTraces on every call.
package pdatahandle

import (
	"runtime"

	"github.com/otelwasm/otelwasm/guest/internal/mem"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Field IDs are OTLP protobuf field numbers, shared with the host side.
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

	fieldResourceAttributes     = 1
	fieldResourceDroppedAttribs = 2

	fieldScopeName           = 1
	fieldScopeVersion        = 2
	fieldScopeAttributes     = 3
	fieldScopeDroppedAttribs = 4
)

func getStr(h, field uint32) string {
	return mem.GetString(func(ptr uint32, limit mem.BufLimit) uint32 {
		return pdataGetStr(h, field, ptr, limit)
	})
}

func setStr(h, field uint32, s string) {
	ptr, size := mem.StringToPtr(s)
	pdataSetStr(h, field, ptr, size)
	runtime.KeepAlive(s) // until ptr is no longer needed
}

// Traces is a handle to the ptrace.Traces held by the host.
type Traces struct{ h uint32 }

// CurrentTraces returns a handle to the traces being processed.
func CurrentTraces() Traces {
	return Traces{h: pdataCurrentTraces()}
}

// Commit tells the host to use the current (possibly mutated in place)
// traces as the result of this call, skipping serialization entirely.
func Commit() {
	pdataCommitCurrentTraces()
}

func (t Traces) ResourceSpans() ResourceSpansSlice {
	return ResourceSpansSlice{h: pdataGetMsg(t.h, fieldTracesResourceSpans)}
}

type ResourceSpansSlice struct{ h uint32 }

func (s ResourceSpansSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s ResourceSpansSlice) At(i int) ResourceSpans {
	return ResourceSpans{h: pdataSliceGet(s.h, uint32(i))}
}

type ResourceSpans struct{ h uint32 }

func (rs ResourceSpans) Resource() Resource {
	return Resource{h: pdataGetMsg(rs.h, fieldResourceSpansResource)}
}

func (rs ResourceSpans) ScopeSpans() ScopeSpansSlice {
	return ScopeSpansSlice{h: pdataGetMsg(rs.h, fieldResourceSpansScopeSpans)}
}

func (rs ResourceSpans) SchemaUrl() string     { return getStr(rs.h, fieldResourceSpansSchemaUrl) }
func (rs ResourceSpans) SetSchemaUrl(v string) { setStr(rs.h, fieldResourceSpansSchemaUrl, v) }

type ScopeSpansSlice struct{ h uint32 }

func (s ScopeSpansSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s ScopeSpansSlice) At(i int) ScopeSpans {
	return ScopeSpans{h: pdataSliceGet(s.h, uint32(i))}
}

type ScopeSpans struct{ h uint32 }

func (ss ScopeSpans) Scope() InstrumentationScope {
	return InstrumentationScope{h: pdataGetMsg(ss.h, fieldScopeSpansScope)}
}

func (ss ScopeSpans) Spans() SpanSlice {
	return SpanSlice{h: pdataGetMsg(ss.h, fieldScopeSpansSpans)}
}

func (ss ScopeSpans) SchemaUrl() string     { return getStr(ss.h, fieldScopeSpansSchemaUrl) }
func (ss ScopeSpans) SetSchemaUrl(v string) { setStr(ss.h, fieldScopeSpansSchemaUrl, v) }

type SpanSlice struct{ h uint32 }

func (s SpanSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s SpanSlice) At(i int) Span {
	return Span{h: pdataSliceGet(s.h, uint32(i))}
}

type Span struct{ h uint32 }

func (s Span) Name() string     { return getStr(s.h, fieldSpanName) }
func (s Span) SetName(v string) { setStr(s.h, fieldSpanName, v) }

func (s Span) TraceState() string     { return getStr(s.h, fieldSpanTraceState) }
func (s Span) SetTraceState(v string) { setStr(s.h, fieldSpanTraceState, v) }

func (s Span) Kind() ptrace.SpanKind     { return ptrace.SpanKind(pdataGetI64(s.h, fieldSpanKind)) }
func (s Span) SetKind(v ptrace.SpanKind) { pdataSetI64(s.h, fieldSpanKind, int64(v)) }

func (s Span) StartTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(s.h, fieldSpanStartTime))
}

func (s Span) SetStartTimestamp(v pcommon.Timestamp) {
	pdataSetI64(s.h, fieldSpanStartTime, int64(v))
}

func (s Span) EndTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(s.h, fieldSpanEndTime))
}

func (s Span) SetEndTimestamp(v pcommon.Timestamp) {
	pdataSetI64(s.h, fieldSpanEndTime, int64(v))
}

func (s Span) DroppedAttributesCount() uint32 {
	return uint32(pdataGetI64(s.h, fieldSpanDroppedAttribs))
}

func (s Span) Flags() uint32     { return uint32(pdataGetI64(s.h, fieldSpanFlags)) }
func (s Span) SetFlags(v uint32) { pdataSetI64(s.h, fieldSpanFlags, int64(v)) }

func (s Span) Attributes() Map {
	return Map{h: pdataGetMsg(s.h, fieldSpanAttributes)}
}

func (s Span) Status() Status {
	return Status{h: pdataGetMsg(s.h, fieldSpanStatus)}
}

type Status struct{ h uint32 }

func (s Status) Code() ptrace.StatusCode     { return ptrace.StatusCode(pdataGetI64(s.h, fieldStatusCode)) }
func (s Status) SetCode(v ptrace.StatusCode) { pdataSetI64(s.h, fieldStatusCode, int64(v)) }
func (s Status) Message() string             { return getStr(s.h, fieldStatusMessage) }
func (s Status) SetMessage(v string)         { setStr(s.h, fieldStatusMessage, v) }

type Resource struct{ h uint32 }

func (r Resource) Attributes() Map {
	return Map{h: pdataGetMsg(r.h, fieldResourceAttributes)}
}

func (r Resource) DroppedAttributesCount() uint32 {
	return uint32(pdataGetI64(r.h, fieldResourceDroppedAttribs))
}

type InstrumentationScope struct{ h uint32 }

func (is InstrumentationScope) Name() string     { return getStr(is.h, fieldScopeName) }
func (is InstrumentationScope) SetName(v string) { setStr(is.h, fieldScopeName, v) }
func (is InstrumentationScope) Version() string  { return getStr(is.h, fieldScopeVersion) }
func (is InstrumentationScope) SetVersion(v string) {
	setStr(is.h, fieldScopeVersion, v)
}

func (is InstrumentationScope) Attributes() Map {
	return Map{h: pdataGetMsg(is.h, fieldScopeAttributes)}
}

// Map is a handle to a pcommon.Map held by the host.
type Map struct{ h uint32 }

func (m Map) Len() int { return int(pdataMapLen(m.h)) }

// GetStr returns the value for key converted to a string, or "" if the key
// is missing (indistinguishable from an empty string value).
func (m Map) GetStr(key string) string {
	keyPtr, keySize := mem.StringToPtr(key)
	v := mem.GetString(func(ptr uint32, limit mem.BufLimit) uint32 {
		return pdataMapGetStr(m.h, keyPtr, keySize, ptr, limit)
	})
	runtime.KeepAlive(key) // until keyPtr is no longer needed
	return v
}

// PutStr upserts a string value for key.
func (m Map) PutStr(key, val string) {
	keyPtr, keySize := mem.StringToPtr(key)
	valPtr, valSize := mem.StringToPtr(val)
	pdataMapPutStr(m.h, keyPtr, keySize, valPtr, valSize)
	runtime.KeepAlive(key) // until keyPtr is no longer needed
	runtime.KeepAlive(val) // until valPtr is no longer needed
}
