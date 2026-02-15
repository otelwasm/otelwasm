package metricsprocessor

import (
	"runtime"

	"github.com/otelwasm/otelwasm/guest/api"
	pubimports "github.com/otelwasm/otelwasm/guest/imports"
	"github.com/otelwasm/otelwasm/guest/internal/imports"
	"github.com/otelwasm/otelwasm/guest/internal/mem"
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

var _ func(uint32, uint32) uint32 = _consumeMetrics

//go:wasmexport consume_metrics
func _consumeMetrics(dataPtr uint32, dataSize uint32) uint32 {
	raw := mem.TakeOwnership(dataPtr, dataSize)
	unmarshaler := pmetric.ProtoUnmarshaler{}
	metrics, err := unmarshaler.UnmarshalMetrics(raw)
	if err != nil {
		return imports.StatusToCode(api.StatusError(err.Error()))
	}

	result, status := metricsprocessor.ProcessMetrics(metrics)
	// If the result is not empty, set it in the host.
	// In case of empty result, the result should be written inside the guest call.
	if result != (pmetric.Metrics{}) {
		pubimports.SetResultMetrics(result)
	}
	runtime.KeepAlive(result) // until ptr is no longer needed

	return imports.StatusToCode(status)
}
