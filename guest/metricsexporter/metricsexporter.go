package metricsexporter

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/plugin"
)

var metricsexporter api.MetricsExporter

func SetPlugin(tp api.MetricsExporter) {
	if tp == nil {
		panic("nil MetricsExporter")
	}
	metricsexporter = tp
	plugin.MustSet(tp)
}

var _ func() uint32 = _pushMetrics

//go:wasmexport pushMetrics
func _pushMetrics() uint32 {
	metrics := imports.CurrentMetrics()
	status := metricsexporter.PushMetrics(metrics)
	return imports.StatusToCode(status)
}
