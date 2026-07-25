package pdatahandle

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// Field IDs are OTLP protobuf field numbers, shared with the host side.
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
	// the data oneof.
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

// Metrics is a handle to the pmetric.Metrics held by the host.
type Metrics struct{ h uint32 }

// CurrentMetrics returns a handle to the metrics being processed.
func CurrentMetrics() Metrics {
	return Metrics{h: pdataCurrentMetrics()}
}

// CommitMetrics tells the host to use the current (possibly mutated in
// place) metrics as the result of this call, skipping serialization
// entirely.
func CommitMetrics() {
	pdataCommitCurrentMetrics()
}

func (m Metrics) ResourceMetrics() ResourceMetricsSlice {
	return ResourceMetricsSlice{h: pdataGetMsg(m.h, fieldMetricsResourceMetrics)}
}

type ResourceMetricsSlice struct{ h uint32 }

func (s ResourceMetricsSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s ResourceMetricsSlice) At(i int) ResourceMetrics {
	return ResourceMetrics{h: pdataSliceGet(s.h, uint32(i))}
}

type ResourceMetrics struct{ h uint32 }

func (rm ResourceMetrics) Resource() Resource {
	return Resource{h: pdataGetMsg(rm.h, fieldResourceMetricsResource)}
}

func (rm ResourceMetrics) ScopeMetrics() ScopeMetricsSlice {
	return ScopeMetricsSlice{h: pdataGetMsg(rm.h, fieldResourceMetricsScopeMetrics)}
}

func (rm ResourceMetrics) SchemaUrl() string     { return getStr(rm.h, fieldResourceMetricsSchemaUrl) }
func (rm ResourceMetrics) SetSchemaUrl(v string) { setStr(rm.h, fieldResourceMetricsSchemaUrl, v) }

type ScopeMetricsSlice struct{ h uint32 }

func (s ScopeMetricsSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s ScopeMetricsSlice) At(i int) ScopeMetrics {
	return ScopeMetrics{h: pdataSliceGet(s.h, uint32(i))}
}

type ScopeMetrics struct{ h uint32 }

func (sm ScopeMetrics) Scope() InstrumentationScope {
	return InstrumentationScope{h: pdataGetMsg(sm.h, fieldScopeMetricsScope)}
}

func (sm ScopeMetrics) Metrics() MetricSlice {
	return MetricSlice{h: pdataGetMsg(sm.h, fieldScopeMetricsMetrics)}
}

func (sm ScopeMetrics) SchemaUrl() string     { return getStr(sm.h, fieldScopeMetricsSchemaUrl) }
func (sm ScopeMetrics) SetSchemaUrl(v string) { setStr(sm.h, fieldScopeMetricsSchemaUrl, v) }

type MetricSlice struct{ h uint32 }

func (s MetricSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s MetricSlice) At(i int) Metric {
	return Metric{h: pdataSliceGet(s.h, uint32(i))}
}

type Metric struct{ h uint32 }

func (m Metric) Name() string            { return getStr(m.h, fieldMetricName) }
func (m Metric) SetName(v string)        { setStr(m.h, fieldMetricName, v) }
func (m Metric) Description() string     { return getStr(m.h, fieldMetricDescription) }
func (m Metric) SetDescription(v string) { setStr(m.h, fieldMetricDescription, v) }
func (m Metric) Unit() string            { return getStr(m.h, fieldMetricUnit) }
func (m Metric) SetUnit(v string)        { setStr(m.h, fieldMetricUnit, v) }

// Type returns the discriminator of the data oneof. Fetching a data field
// whose type doesn't match it panics on the host side.
func (m Metric) Type() pmetric.MetricType {
	return pmetric.MetricType(pdataGetI64(m.h, fieldMetricDataType))
}

func (m Metric) Gauge() Gauge {
	return Gauge{h: pdataGetMsg(m.h, fieldMetricGauge)}
}

func (m Metric) Sum() Sum {
	return Sum{h: pdataGetMsg(m.h, fieldMetricSum)}
}

func (m Metric) Histogram() Histogram {
	return Histogram{h: pdataGetMsg(m.h, fieldMetricHistogram)}
}

func (m Metric) ExponentialHistogram() ExponentialHistogram {
	return ExponentialHistogram{h: pdataGetMsg(m.h, fieldMetricExponentialHistogram)}
}

func (m Metric) Summary() Summary {
	return Summary{h: pdataGetMsg(m.h, fieldMetricSummary)}
}

type Gauge struct{ h uint32 }

func (g Gauge) DataPoints() NumberDataPointSlice {
	return NumberDataPointSlice{h: pdataGetMsg(g.h, fieldGaugeDataPoints)}
}

type Sum struct{ h uint32 }

func (s Sum) DataPoints() NumberDataPointSlice {
	return NumberDataPointSlice{h: pdataGetMsg(s.h, fieldSumDataPoints)}
}

func (s Sum) AggregationTemporality() pmetric.AggregationTemporality {
	return pmetric.AggregationTemporality(pdataGetI64(s.h, fieldSumAggregationTemporality))
}

func (s Sum) SetAggregationTemporality(v pmetric.AggregationTemporality) {
	pdataSetI64(s.h, fieldSumAggregationTemporality, int64(v))
}

func (s Sum) IsMonotonic() bool {
	return pdataGetI64(s.h, fieldSumIsMonotonic) != 0
}

func (s Sum) SetIsMonotonic(v bool) {
	var i int64
	if v {
		i = 1
	}
	pdataSetI64(s.h, fieldSumIsMonotonic, i)
}

type Histogram struct{ h uint32 }

func (h Histogram) DataPoints() HistogramDataPointSlice {
	return HistogramDataPointSlice{h: pdataGetMsg(h.h, fieldHistogramDataPoints)}
}

func (h Histogram) AggregationTemporality() pmetric.AggregationTemporality {
	return pmetric.AggregationTemporality(pdataGetI64(h.h, fieldHistogramAggregationTemporality))
}

type ExponentialHistogram struct{ h uint32 }

func (h ExponentialHistogram) DataPoints() ExponentialHistogramDataPointSlice {
	return ExponentialHistogramDataPointSlice{h: pdataGetMsg(h.h, fieldExponentialHistogramDataPoints)}
}

func (h ExponentialHistogram) AggregationTemporality() pmetric.AggregationTemporality {
	return pmetric.AggregationTemporality(pdataGetI64(h.h, fieldExponentialHistogramAggregationTemporality))
}

type Summary struct{ h uint32 }

func (s Summary) DataPoints() SummaryDataPointSlice {
	return SummaryDataPointSlice{h: pdataGetMsg(s.h, fieldSummaryDataPoints)}
}

type NumberDataPointSlice struct{ h uint32 }

func (s NumberDataPointSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s NumberDataPointSlice) At(i int) NumberDataPoint {
	return NumberDataPoint{h: pdataSliceGet(s.h, uint32(i))}
}

type NumberDataPoint struct{ h uint32 }

// ValueType returns the discriminator of the value oneof.
func (p NumberDataPoint) ValueType() pmetric.NumberDataPointValueType {
	return pmetric.NumberDataPointValueType(pdataGetI64(p.h, fieldNumberDataPointValueType))
}

func (p NumberDataPoint) StartTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(p.h, fieldNumberDataPointStartTime))
}

func (p NumberDataPoint) SetStartTimestamp(v pcommon.Timestamp) {
	pdataSetI64(p.h, fieldNumberDataPointStartTime, int64(v))
}

func (p NumberDataPoint) Timestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(p.h, fieldNumberDataPointTime))
}

func (p NumberDataPoint) SetTimestamp(v pcommon.Timestamp) {
	pdataSetI64(p.h, fieldNumberDataPointTime, int64(v))
}

func (p NumberDataPoint) DoubleValue() float64 {
	return pdataGetF64(p.h, fieldNumberDataPointAsDouble)
}

func (p NumberDataPoint) SetDoubleValue(v float64) {
	pdataSetF64(p.h, fieldNumberDataPointAsDouble, v)
}

func (p NumberDataPoint) IntValue() int64 {
	return pdataGetI64(p.h, fieldNumberDataPointAsInt)
}

func (p NumberDataPoint) SetIntValue(v int64) {
	pdataSetI64(p.h, fieldNumberDataPointAsInt, v)
}

func (p NumberDataPoint) Flags() uint32 {
	return uint32(pdataGetI64(p.h, fieldNumberDataPointFlags))
}

func (p NumberDataPoint) Attributes() Map {
	return Map{h: pdataGetMsg(p.h, fieldNumberDataPointAttribs)}
}

type HistogramDataPointSlice struct{ h uint32 }

func (s HistogramDataPointSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s HistogramDataPointSlice) At(i int) HistogramDataPoint {
	return HistogramDataPoint{h: pdataSliceGet(s.h, uint32(i))}
}

type HistogramDataPoint struct{ h uint32 }

func (p HistogramDataPoint) StartTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(p.h, fieldHistogramDataPointStartTime))
}

func (p HistogramDataPoint) Timestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(p.h, fieldHistogramDataPointTime))
}

func (p HistogramDataPoint) Count() uint64 {
	return uint64(pdataGetI64(p.h, fieldHistogramDataPointCount))
}

func (p HistogramDataPoint) Sum() float64     { return pdataGetF64(p.h, fieldHistogramDataPointSum) }
func (p HistogramDataPoint) SetSum(v float64) { pdataSetF64(p.h, fieldHistogramDataPointSum, v) }
func (p HistogramDataPoint) Min() float64     { return pdataGetF64(p.h, fieldHistogramDataPointMin) }
func (p HistogramDataPoint) Max() float64     { return pdataGetF64(p.h, fieldHistogramDataPointMax) }

func (p HistogramDataPoint) Attributes() Map {
	return Map{h: pdataGetMsg(p.h, fieldHistogramDataPointAttribs)}
}

type ExponentialHistogramDataPointSlice struct{ h uint32 }

func (s ExponentialHistogramDataPointSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s ExponentialHistogramDataPointSlice) At(i int) ExponentialHistogramDataPoint {
	return ExponentialHistogramDataPoint{h: pdataSliceGet(s.h, uint32(i))}
}

type ExponentialHistogramDataPoint struct{ h uint32 }

func (p ExponentialHistogramDataPoint) StartTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(p.h, fieldExponentialHistogramDataPointStartTime))
}

func (p ExponentialHistogramDataPoint) Timestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(p.h, fieldExponentialHistogramDataPointTime))
}

func (p ExponentialHistogramDataPoint) Count() uint64 {
	return uint64(pdataGetI64(p.h, fieldExponentialHistogramDataPointCount))
}

func (p ExponentialHistogramDataPoint) Sum() float64 {
	return pdataGetF64(p.h, fieldExponentialHistogramDataPointSum)
}

func (p ExponentialHistogramDataPoint) Scale() int32 {
	return int32(pdataGetI64(p.h, fieldExponentialHistogramDataPointScale))
}

func (p ExponentialHistogramDataPoint) ZeroCount() uint64 {
	return uint64(pdataGetI64(p.h, fieldExponentialHistogramDataPointZeroCount))
}

func (p ExponentialHistogramDataPoint) Min() float64 {
	return pdataGetF64(p.h, fieldExponentialHistogramDataPointMin)
}

func (p ExponentialHistogramDataPoint) Max() float64 {
	return pdataGetF64(p.h, fieldExponentialHistogramDataPointMax)
}

func (p ExponentialHistogramDataPoint) Attributes() Map {
	return Map{h: pdataGetMsg(p.h, fieldExponentialHistogramDataPointAttribs)}
}

type SummaryDataPointSlice struct{ h uint32 }

func (s SummaryDataPointSlice) Len() int { return int(pdataSliceLen(s.h)) }
func (s SummaryDataPointSlice) At(i int) SummaryDataPoint {
	return SummaryDataPoint{h: pdataSliceGet(s.h, uint32(i))}
}

type SummaryDataPoint struct{ h uint32 }

func (p SummaryDataPoint) StartTimestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(p.h, fieldSummaryDataPointStartTime))
}

func (p SummaryDataPoint) Timestamp() pcommon.Timestamp {
	return pcommon.Timestamp(pdataGetI64(p.h, fieldSummaryDataPointTime))
}

func (p SummaryDataPoint) Count() uint64 {
	return uint64(pdataGetI64(p.h, fieldSummaryDataPointCount))
}

func (p SummaryDataPoint) Sum() float64     { return pdataGetF64(p.h, fieldSummaryDataPointSum) }
func (p SummaryDataPoint) SetSum(v float64) { pdataSetF64(p.h, fieldSummaryDataPointSum, v) }

func (p SummaryDataPoint) Attributes() Map {
	return Map{h: pdataGetMsg(p.h, fieldSummaryDataPointAttribs)}
}
