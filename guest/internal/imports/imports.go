package imports

import "github.com/musaprg/otelwasm/guest/internal/mem"

//go:wasmimport opentelemetry.io/wasm currentTraces
func currentTraces(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm currentMetrics
func currentMetrics(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm currentLogs
func currentLogs(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm setResultTraces
func setResultTraces(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm setResultMetrics
func setResultMetrics(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm setResultLogs
func setResultLogs(ptr, size uint32)
