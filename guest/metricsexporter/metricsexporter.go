package metricsexporter

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/mem"
	"github.com/otelwasm/otelwasm/guest/internal/plugin"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var metricsexporter api.MetricsExporter

func SetPlugin(tp api.MetricsExporter) {
	if tp == nil {
		panic("nil MetricsExporter")
	}
	metricsexporter = tp
	plugin.MustSet(tp)
}

var _ func(uint32, uint32) uint32 = _consumeMetrics

//go:wasmexport otelwasm_consume_metrics
func _consumeMetrics(dataPtr uint32, dataSize uint32) uint32 {
	raw := mem.TakeOwnership(dataPtr, dataSize)
	unmarshaler := pmetric.ProtoUnmarshaler{}
	metrics, err := unmarshaler.UnmarshalMetrics(raw)
	if err != nil {
		return imports.StatusToCode(api.StatusError(err.Error()))
	}

	status := metricsexporter.PushMetrics(metrics)
	return imports.StatusToCode(status)
}
