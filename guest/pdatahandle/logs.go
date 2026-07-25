package pdatahandle

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

// Field IDs are OTLP protobuf field numbers, shared with the host side.
const (
	fieldLogsResourceLogs = 1

	fieldResourceLogsResource  = 1
	fieldResourceLogsScopeLogs = 2
	fieldResourceLogsSchemaUrl = 3

	fieldScopeLogsScope      = 1
	fieldScopeLogsLogRecords = 2
	fieldScopeLogsSchemaUrl  = 3

	fieldLogRecordTime           = 1
	fieldLogRecordSeverityNumber = 2
	fieldLogRecordSeverityText   = 3
	// fieldLogRecordBody is exposed as a string: reads return
	// Body().AsString() and writes call Body().SetStr().
	fieldLogRecordBody           = 5
	fieldLogRecordAttributes     = 6
	fieldLogRecordDroppedAttribs = 7
	fieldLogRecordFlags          = 8
	fieldLogRecordObservedTime   = 11
	fieldLogRecordEventName      = 12
)

// Logs is a handle to the plog.Logs held by the host.
type Logs struct{ h uint32 }

// CurrentLogs returns a handle to the logs being processed.
func CurrentLogs() Logs {
	return Logs{h: pdataCurrentLogs()}
}

// CommitLogs tells the host to use the current (possibly mutated in place)
// logs as the result of this call, skipping serialization entirely.
func CommitLogs() {
	pdataCommitCurrentLogs()
}

func (l Logs) ResourceLogs() ResourceLogsSlice {
	return ResourceLogsSlice{h: pdataGetMsg(l.h, fieldLogsResourceLogs)}
}

type ResourceLogsSlice struct{ h uint32 }

func (s ResourceLogsSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s ResourceLogsSlice) At(i int) ResourceLogs {
	return ResourceLogs{h: pdataSliceGet(s.h, uint32(i))}
}

type ResourceLogs struct{ h uint32 }

func (rl ResourceLogs) Resource() Resource {
	return Resource{h: pdataGetMsg(rl.h, fieldResourceLogsResource)}
}

func (rl ResourceLogs) ScopeLogs() ScopeLogsSlice {
	return ScopeLogsSlice{h: pdataGetMsg(rl.h, fieldResourceLogsScopeLogs)}
}

func (rl ResourceLogs) SchemaUrl() string     { return getStr(rl.h, fieldResourceLogsSchemaUrl) }
func (rl ResourceLogs) SetSchemaUrl(v string) { setStr(rl.h, fieldResourceLogsSchemaUrl, v) }

type ScopeLogsSlice struct{ h uint32 }

func (s ScopeLogsSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s ScopeLogsSlice) At(i int) ScopeLogs {
	return ScopeLogs{h: pdataSliceGet(s.h, uint32(i))}
}

type ScopeLogs struct{ h uint32 }

func (sl ScopeLogs) Scope() InstrumentationScope {
	return InstrumentationScope{h: pdataGetMsg(sl.h, fieldScopeLogsScope)}
}

func (sl ScopeLogs) LogRecords() LogRecordSlice {
	return LogRecordSlice{h: pdataGetMsg(sl.h, fieldScopeLogsLogRecords)}
}

func (sl ScopeLogs) SchemaUrl() string     { return getStr(sl.h, fieldScopeLogsSchemaUrl) }
func (sl ScopeLogs) SetSchemaUrl(v string) { setStr(sl.h, fieldScopeLogsSchemaUrl, v) }

type LogRecordSlice struct{ h uint32 }

func (s LogRecordSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s LogRecordSlice) At(i int) LogRecord {
	return LogRecord{h: pdataSliceGet(s.h, uint32(i))}
}

type LogRecord struct{ h uint32 }

func (lr LogRecord) Timestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(lr.h, fieldLogRecordTime))
}

func (lr LogRecord) SetTimestamp(v pcommon.Timestamp) {
	pdataSetI64(lr.h, fieldLogRecordTime, int64(v))
}

func (lr LogRecord) ObservedTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(lr.h, fieldLogRecordObservedTime))
}

func (lr LogRecord) SetObservedTimestamp(v pcommon.Timestamp) {
	pdataSetI64(lr.h, fieldLogRecordObservedTime, int64(v))
}

func (lr LogRecord) SeverityNumber() plog.SeverityNumber {
	return plog.SeverityNumber(pdataGetI64(lr.h, fieldLogRecordSeverityNumber))
}

func (lr LogRecord) SetSeverityNumber(v plog.SeverityNumber) {
	pdataSetI64(lr.h, fieldLogRecordSeverityNumber, int64(v))
}

func (lr LogRecord) SeverityText() string     { return getStr(lr.h, fieldLogRecordSeverityText) }
func (lr LogRecord) SetSeverityText(v string) { setStr(lr.h, fieldLogRecordSeverityText, v) }

// BodyStr returns the body converted to a string.
func (lr LogRecord) BodyStr() string { return getStr(lr.h, fieldLogRecordBody) }

// SetBodyStr sets the body to a string value.
func (lr LogRecord) SetBodyStr(v string) { setStr(lr.h, fieldLogRecordBody, v) }

func (lr LogRecord) DroppedAttributesCount() uint32 {
	return uint32(pdataGetI64(lr.h, fieldLogRecordDroppedAttribs))
}

func (lr LogRecord) Flags() uint32     { return uint32(pdataGetI64(lr.h, fieldLogRecordFlags)) }
func (lr LogRecord) SetFlags(v uint32) { pdataSetI64(lr.h, fieldLogRecordFlags, int64(v)) }

func (lr LogRecord) EventName() string     { return getStr(lr.h, fieldLogRecordEventName) }
func (lr LogRecord) SetEventName(v string) { setStr(lr.h, fieldLogRecordEventName, v) }

func (lr LogRecord) Attributes() Map {
	return Map{h: pdataGetMsg(lr.h, fieldLogRecordAttributes)}
}
