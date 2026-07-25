package wasmplugin

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero/api"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
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
	pdataCurrentTraces        = "pdataCurrentTraces"
	pdataCommitCurrentTraces  = "pdataCommitCurrentTraces"
	pdataCurrentMetrics       = "pdataCurrentMetrics"
	pdataCommitCurrentMetrics = "pdataCommitCurrentMetrics"
	pdataCurrentLogs          = "pdataCurrentLogs"
	pdataCommitCurrentLogs    = "pdataCommitCurrentLogs"
	pdataGetI64               = "pdataGetI64"
	pdataSetI64               = "pdataSetI64"
	pdataGetF64               = "pdataGetF64"
	pdataSetF64               = "pdataSetF64"
	pdataGetStr               = "pdataGetStr"
	pdataSetStr               = "pdataSetStr"
	pdataGetMsg               = "pdataGetMsg"
	pdataSliceLen             = "pdataSliceLen"
	pdataSliceGet             = "pdataSliceGet"
	pdataMapLen               = "pdataMapLen"
	pdataMapGetStr            = "pdataMapGetStr"
	pdataMapPutStr            = "pdataMapPutStr"
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

// https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/metrics/v1/metrics.proto
const (
	fieldMetricsResourceMetrics = 1

	fieldResourceMetricsResource     = 1
	fieldResourceMetricsScopeMetrics = 2
	fieldResourceMetricsSchemaUrl    = 3

	fieldScopeMetricsScope     = 1
	fieldScopeMetricsMetrics   = 2
	fieldScopeMetricsSchemaUrl = 3

	// fieldMetricDataType is a virtual field (proto field numbers start at
	// 1): reading it as i64 returns the pmetric.MetricType discriminator of
	// the data oneof. Guests must check it before fetching the oneof field.
	fieldMetricDataType             = 0
	fieldMetricName                 = 1
	fieldMetricDescription          = 2
	fieldMetricUnit                 = 3
	fieldMetricGauge                = 5
	fieldMetricSum                  = 7
	fieldMetricHistogram            = 9
	fieldMetricExponentialHistogram = 10
	fieldMetricSummary              = 11

	fieldGaugeDataPoints = 1

	fieldSumDataPoints             = 1
	fieldSumAggregationTemporality = 2
	fieldSumIsMonotonic            = 3

	fieldHistogramDataPoints             = 1
	fieldHistogramAggregationTemporality = 2

	fieldExponentialHistogramDataPoints             = 1
	fieldExponentialHistogramAggregationTemporality = 2

	fieldSummaryDataPoints = 1

	// fieldNumberDataPointValueType is a virtual field: reading it as i64
	// returns the pmetric.NumberDataPointValueType discriminator of the
	// value oneof.
	fieldNumberDataPointValueType = 0
	fieldNumberDataPointStartTime = 2
	fieldNumberDataPointTime      = 3
	fieldNumberDataPointAsDouble  = 4
	fieldNumberDataPointAsInt     = 6
	fieldNumberDataPointAttribs   = 7
	fieldNumberDataPointFlags     = 8

	fieldHistogramDataPointStartTime = 2
	fieldHistogramDataPointTime      = 3
	fieldHistogramDataPointCount     = 4
	fieldHistogramDataPointSum       = 5
	fieldHistogramDataPointAttribs   = 9
	fieldHistogramDataPointFlags     = 10
	fieldHistogramDataPointMin       = 11
	fieldHistogramDataPointMax       = 12

	fieldExponentialHistogramDataPointAttribs   = 1
	fieldExponentialHistogramDataPointStartTime = 2
	fieldExponentialHistogramDataPointTime      = 3
	fieldExponentialHistogramDataPointCount     = 4
	fieldExponentialHistogramDataPointSum       = 5
	fieldExponentialHistogramDataPointScale     = 6
	fieldExponentialHistogramDataPointZeroCount = 7
	fieldExponentialHistogramDataPointFlags     = 10
	fieldExponentialHistogramDataPointMin       = 12
	fieldExponentialHistogramDataPointMax       = 13

	fieldSummaryDataPointStartTime = 2
	fieldSummaryDataPointTime      = 3
	fieldSummaryDataPointCount     = 4
	fieldSummaryDataPointSum       = 5
	fieldSummaryDataPointAttribs   = 7
	fieldSummaryDataPointFlags     = 8
)

// https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/logs/v1/logs.proto
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

func pdataCurrentMetricsFn(ctx context.Context, mod api.Module, stack []uint64) {
	s := paramsFromContext(ctx)
	stack[0] = uint64(s.addHandle(s.CurrentMetrics))
}

func pdataCommitCurrentMetricsFn(ctx context.Context, mod api.Module, stack []uint64) {
	s := paramsFromContext(ctx)
	s.ResultMetrics = s.CurrentMetrics
	if s.OnResultMetricsChange != nil {
		s.OnResultMetricsChange(s.ResultMetrics)
	}
}

func pdataCurrentLogsFn(ctx context.Context, mod api.Module, stack []uint64) {
	s := paramsFromContext(ctx)
	stack[0] = uint64(s.addHandle(s.CurrentLogs))
}

func pdataCommitCurrentLogsFn(ctx context.Context, mod api.Module, stack []uint64) {
	s := paramsFromContext(ctx)
	s.ResultLogs = s.CurrentLogs
	if s.OnResultLogsChange != nil {
		s.OnResultLogsChange(s.ResultLogs)
	}
}

func pdataGetF64Fn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	field := uint32(stack[1])

	var v float64
	switch o := paramsFromContext(ctx).getHandle(h).(type) {
	case pmetric.NumberDataPoint:
		switch field {
		case fieldNumberDataPointAsDouble:
			v = o.DoubleValue()
		default:
			panicUnknownField("NumberDataPoint", field)
		}
	case pmetric.HistogramDataPoint:
		switch field {
		case fieldHistogramDataPointSum:
			v = o.Sum()
		case fieldHistogramDataPointMin:
			v = o.Min()
		case fieldHistogramDataPointMax:
			v = o.Max()
		default:
			panicUnknownField("HistogramDataPoint", field)
		}
	case pmetric.ExponentialHistogramDataPoint:
		switch field {
		case fieldExponentialHistogramDataPointSum:
			v = o.Sum()
		case fieldExponentialHistogramDataPointMin:
			v = o.Min()
		case fieldExponentialHistogramDataPointMax:
			v = o.Max()
		default:
			panicUnknownField("ExponentialHistogramDataPoint", field)
		}
	case pmetric.SummaryDataPoint:
		switch field {
		case fieldSummaryDataPointSum:
			v = o.Sum()
		default:
			panicUnknownField("SummaryDataPoint", field)
		}
	default:
		panicUnknownType("pdataGetF64", o)
	}
	stack[0] = api.EncodeF64(v)
}

func pdataSetF64Fn(ctx context.Context, mod api.Module, stack []uint64) {
	h := uint32(stack[0])
	field := uint32(stack[1])
	v := api.DecodeF64(stack[2])

	switch o := paramsFromContext(ctx).getHandle(h).(type) {
	case pmetric.NumberDataPoint:
		switch field {
		case fieldNumberDataPointAsDouble:
			o.SetDoubleValue(v)
		default:
			panicUnknownField("NumberDataPoint", field)
		}
	case pmetric.HistogramDataPoint:
		switch field {
		case fieldHistogramDataPointSum:
			o.SetSum(v)
		case fieldHistogramDataPointMin:
			o.SetMin(v)
		case fieldHistogramDataPointMax:
			o.SetMax(v)
		default:
			panicUnknownField("HistogramDataPoint", field)
		}
	case pmetric.SummaryDataPoint:
		switch field {
		case fieldSummaryDataPointSum:
			o.SetSum(v)
		default:
			panicUnknownField("SummaryDataPoint", field)
		}
	default:
		panicUnknownType("pdataSetF64", o)
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
	case pmetric.Metric:
		switch field {
		case fieldMetricDataType:
			v = int64(o.Type())
		default:
			panicUnknownField("Metric", field)
		}
	case pmetric.Sum:
		switch field {
		case fieldSumAggregationTemporality:
			v = int64(o.AggregationTemporality())
		case fieldSumIsMonotonic:
			if o.IsMonotonic() {
				v = 1
			}
		default:
			panicUnknownField("Sum", field)
		}
	case pmetric.Histogram:
		switch field {
		case fieldHistogramAggregationTemporality:
			v = int64(o.AggregationTemporality())
		default:
			panicUnknownField("Histogram", field)
		}
	case pmetric.ExponentialHistogram:
		switch field {
		case fieldExponentialHistogramAggregationTemporality:
			v = int64(o.AggregationTemporality())
		default:
			panicUnknownField("ExponentialHistogram", field)
		}
	case pmetric.NumberDataPoint:
		switch field {
		case fieldNumberDataPointValueType:
			v = int64(o.ValueType())
		case fieldNumberDataPointStartTime:
			v = int64(o.StartTimestamp())
		case fieldNumberDataPointTime:
			v = int64(o.Timestamp())
		case fieldNumberDataPointAsInt:
			v = o.IntValue()
		case fieldNumberDataPointFlags:
			v = int64(o.Flags())
		default:
			panicUnknownField("NumberDataPoint", field)
		}
	case pmetric.HistogramDataPoint:
		switch field {
		case fieldHistogramDataPointStartTime:
			v = int64(o.StartTimestamp())
		case fieldHistogramDataPointTime:
			v = int64(o.Timestamp())
		case fieldHistogramDataPointCount:
			v = int64(o.Count())
		case fieldHistogramDataPointFlags:
			v = int64(o.Flags())
		default:
			panicUnknownField("HistogramDataPoint", field)
		}
	case pmetric.ExponentialHistogramDataPoint:
		switch field {
		case fieldExponentialHistogramDataPointStartTime:
			v = int64(o.StartTimestamp())
		case fieldExponentialHistogramDataPointTime:
			v = int64(o.Timestamp())
		case fieldExponentialHistogramDataPointCount:
			v = int64(o.Count())
		case fieldExponentialHistogramDataPointScale:
			v = int64(o.Scale())
		case fieldExponentialHistogramDataPointZeroCount:
			v = int64(o.ZeroCount())
		case fieldExponentialHistogramDataPointFlags:
			v = int64(o.Flags())
		default:
			panicUnknownField("ExponentialHistogramDataPoint", field)
		}
	case pmetric.SummaryDataPoint:
		switch field {
		case fieldSummaryDataPointStartTime:
			v = int64(o.StartTimestamp())
		case fieldSummaryDataPointTime:
			v = int64(o.Timestamp())
		case fieldSummaryDataPointCount:
			v = int64(o.Count())
		case fieldSummaryDataPointFlags:
			v = int64(o.Flags())
		default:
			panicUnknownField("SummaryDataPoint", field)
		}
	case plog.LogRecord:
		switch field {
		case fieldLogRecordTime:
			v = int64(o.Timestamp())
		case fieldLogRecordSeverityNumber:
			v = int64(o.SeverityNumber())
		case fieldLogRecordDroppedAttribs:
			v = int64(o.DroppedAttributesCount())
		case fieldLogRecordFlags:
			v = int64(o.Flags())
		case fieldLogRecordObservedTime:
			v = int64(o.ObservedTimestamp())
		default:
			panicUnknownField("LogRecord", field)
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
	case pmetric.Sum:
		switch field {
		case fieldSumAggregationTemporality:
			o.SetAggregationTemporality(pmetric.AggregationTemporality(v))
		case fieldSumIsMonotonic:
			o.SetIsMonotonic(v != 0)
		default:
			panicUnknownField("Sum", field)
		}
	case pmetric.NumberDataPoint:
		switch field {
		case fieldNumberDataPointStartTime:
			o.SetStartTimestamp(pcommon.Timestamp(v))
		case fieldNumberDataPointTime:
			o.SetTimestamp(pcommon.Timestamp(v))
		case fieldNumberDataPointAsInt:
			o.SetIntValue(v)
		case fieldNumberDataPointFlags:
			o.SetFlags(pmetric.DataPointFlags(v))
		default:
			panicUnknownField("NumberDataPoint", field)
		}
	case plog.LogRecord:
		switch field {
		case fieldLogRecordTime:
			o.SetTimestamp(pcommon.Timestamp(v))
		case fieldLogRecordSeverityNumber:
			o.SetSeverityNumber(plog.SeverityNumber(v))
		case fieldLogRecordFlags:
			o.SetFlags(plog.LogRecordFlags(v))
		case fieldLogRecordObservedTime:
			o.SetObservedTimestamp(pcommon.Timestamp(v))
		default:
			panicUnknownField("LogRecord", field)
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
	case pmetric.ResourceMetrics:
		switch field {
		case fieldResourceMetricsSchemaUrl:
			v = o.SchemaUrl()
		default:
			panicUnknownField("ResourceMetrics", field)
		}
	case pmetric.ScopeMetrics:
		switch field {
		case fieldScopeMetricsSchemaUrl:
			v = o.SchemaUrl()
		default:
			panicUnknownField("ScopeMetrics", field)
		}
	case pmetric.Metric:
		switch field {
		case fieldMetricName:
			v = o.Name()
		case fieldMetricDescription:
			v = o.Description()
		case fieldMetricUnit:
			v = o.Unit()
		default:
			panicUnknownField("Metric", field)
		}
	case plog.ResourceLogs:
		switch field {
		case fieldResourceLogsSchemaUrl:
			v = o.SchemaUrl()
		default:
			panicUnknownField("ResourceLogs", field)
		}
	case plog.ScopeLogs:
		switch field {
		case fieldScopeLogsSchemaUrl:
			v = o.SchemaUrl()
		default:
			panicUnknownField("ScopeLogs", field)
		}
	case plog.LogRecord:
		switch field {
		case fieldLogRecordSeverityText:
			v = o.SeverityText()
		case fieldLogRecordBody:
			v = o.Body().AsString()
		case fieldLogRecordEventName:
			v = o.EventName()
		default:
			panicUnknownField("LogRecord", field)
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
	case pmetric.ResourceMetrics:
		switch field {
		case fieldResourceMetricsSchemaUrl:
			o.SetSchemaUrl(v)
		default:
			panicUnknownField("ResourceMetrics", field)
		}
	case pmetric.ScopeMetrics:
		switch field {
		case fieldScopeMetricsSchemaUrl:
			o.SetSchemaUrl(v)
		default:
			panicUnknownField("ScopeMetrics", field)
		}
	case pmetric.Metric:
		switch field {
		case fieldMetricName:
			o.SetName(v)
		case fieldMetricDescription:
			o.SetDescription(v)
		case fieldMetricUnit:
			o.SetUnit(v)
		default:
			panicUnknownField("Metric", field)
		}
	case plog.ResourceLogs:
		switch field {
		case fieldResourceLogsSchemaUrl:
			o.SetSchemaUrl(v)
		default:
			panicUnknownField("ResourceLogs", field)
		}
	case plog.ScopeLogs:
		switch field {
		case fieldScopeLogsSchemaUrl:
			o.SetSchemaUrl(v)
		default:
			panicUnknownField("ScopeLogs", field)
		}
	case plog.LogRecord:
		switch field {
		case fieldLogRecordSeverityText:
			o.SetSeverityText(v)
		case fieldLogRecordBody:
			o.Body().SetStr(v)
		case fieldLogRecordEventName:
			o.SetEventName(v)
		default:
			panicUnknownField("LogRecord", field)
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
	case pmetric.Metrics:
		switch field {
		case fieldMetricsResourceMetrics:
			child = o.ResourceMetrics()
		default:
			panicUnknownField("Metrics", field)
		}
	case pmetric.ResourceMetrics:
		switch field {
		case fieldResourceMetricsResource:
			child = o.Resource()
		case fieldResourceMetricsScopeMetrics:
			child = o.ScopeMetrics()
		default:
			panicUnknownField("ResourceMetrics", field)
		}
	case pmetric.ScopeMetrics:
		switch field {
		case fieldScopeMetricsScope:
			child = o.Scope()
		case fieldScopeMetricsMetrics:
			child = o.Metrics()
		default:
			panicUnknownField("ScopeMetrics", field)
		}
	case pmetric.Metric:
		// Fetching a oneof field whose type doesn't match the
		// discriminator (fieldMetricDataType) panics in pdata.
		switch field {
		case fieldMetricGauge:
			child = o.Gauge()
		case fieldMetricSum:
			child = o.Sum()
		case fieldMetricHistogram:
			child = o.Histogram()
		case fieldMetricExponentialHistogram:
			child = o.ExponentialHistogram()
		case fieldMetricSummary:
			child = o.Summary()
		default:
			panicUnknownField("Metric", field)
		}
	case pmetric.Gauge:
		switch field {
		case fieldGaugeDataPoints:
			child = o.DataPoints()
		default:
			panicUnknownField("Gauge", field)
		}
	case pmetric.Sum:
		switch field {
		case fieldSumDataPoints:
			child = o.DataPoints()
		default:
			panicUnknownField("Sum", field)
		}
	case pmetric.Histogram:
		switch field {
		case fieldHistogramDataPoints:
			child = o.DataPoints()
		default:
			panicUnknownField("Histogram", field)
		}
	case pmetric.ExponentialHistogram:
		switch field {
		case fieldExponentialHistogramDataPoints:
			child = o.DataPoints()
		default:
			panicUnknownField("ExponentialHistogram", field)
		}
	case pmetric.Summary:
		switch field {
		case fieldSummaryDataPoints:
			child = o.DataPoints()
		default:
			panicUnknownField("Summary", field)
		}
	case pmetric.NumberDataPoint:
		switch field {
		case fieldNumberDataPointAttribs:
			child = o.Attributes()
		default:
			panicUnknownField("NumberDataPoint", field)
		}
	case pmetric.HistogramDataPoint:
		switch field {
		case fieldHistogramDataPointAttribs:
			child = o.Attributes()
		default:
			panicUnknownField("HistogramDataPoint", field)
		}
	case pmetric.ExponentialHistogramDataPoint:
		switch field {
		case fieldExponentialHistogramDataPointAttribs:
			child = o.Attributes()
		default:
			panicUnknownField("ExponentialHistogramDataPoint", field)
		}
	case pmetric.SummaryDataPoint:
		switch field {
		case fieldSummaryDataPointAttribs:
			child = o.Attributes()
		default:
			panicUnknownField("SummaryDataPoint", field)
		}
	case plog.Logs:
		switch field {
		case fieldLogsResourceLogs:
			child = o.ResourceLogs()
		default:
			panicUnknownField("Logs", field)
		}
	case plog.ResourceLogs:
		switch field {
		case fieldResourceLogsResource:
			child = o.Resource()
		case fieldResourceLogsScopeLogs:
			child = o.ScopeLogs()
		default:
			panicUnknownField("ResourceLogs", field)
		}
	case plog.ScopeLogs:
		switch field {
		case fieldScopeLogsScope:
			child = o.Scope()
		case fieldScopeLogsLogRecords:
			child = o.LogRecords()
		default:
			panicUnknownField("ScopeLogs", field)
		}
	case plog.LogRecord:
		switch field {
		case fieldLogRecordAttributes:
			child = o.Attributes()
		default:
			panicUnknownField("LogRecord", field)
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
	case pmetric.ResourceMetricsSlice:
		n = o.Len()
	case pmetric.ScopeMetricsSlice:
		n = o.Len()
	case pmetric.MetricSlice:
		n = o.Len()
	case pmetric.NumberDataPointSlice:
		n = o.Len()
	case pmetric.HistogramDataPointSlice:
		n = o.Len()
	case pmetric.ExponentialHistogramDataPointSlice:
		n = o.Len()
	case pmetric.SummaryDataPointSlice:
		n = o.Len()
	case plog.ResourceLogsSlice:
		n = o.Len()
	case plog.ScopeLogsSlice:
		n = o.Len()
	case plog.LogRecordSlice:
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
	case pmetric.ResourceMetricsSlice:
		elem = o.At(index)
	case pmetric.ScopeMetricsSlice:
		elem = o.At(index)
	case pmetric.MetricSlice:
		elem = o.At(index)
	case pmetric.NumberDataPointSlice:
		elem = o.At(index)
	case pmetric.HistogramDataPointSlice:
		elem = o.At(index)
	case pmetric.ExponentialHistogramDataPointSlice:
		elem = o.At(index)
	case pmetric.SummaryDataPointSlice:
		elem = o.At(index)
	case plog.ResourceLogsSlice:
		elem = o.At(index)
	case plog.ScopeLogsSlice:
		elem = o.At(index)
	case plog.LogRecordSlice:
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
