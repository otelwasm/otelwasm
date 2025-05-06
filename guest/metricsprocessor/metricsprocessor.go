package metricsprocessor

import (
	"runtime"

	"github.com/otelwasm/otelwasm/guest/api"
	pubimports "github.com/otelwasm/otelwasm/guest/imports"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/plugin"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var metricsprocessor api.MetricsProcessor

func SetPlugin(mp api.MetricsProcessor) {
	if mp == nil {
		panic("nil MetricsProcessor")
	}
	metricsprocessor = mp
	plugin.MustSet(mp)
}

var _ func() uint32 = _processMetrics

//go:wasmexport processMetrics
func _processMetrics() uint32 {
	metrics := imports.CurrentMetrics()
	result, status := metricsprocessor.ProcessMetrics(metrics)
	if result == (pmetric.Metrics{}) {
		pubimports.SetResultMetrics(result)
	}
	runtime.KeepAlive(result) // until ptr is no longer needed

	return imports.StatusToCode(status)
}
