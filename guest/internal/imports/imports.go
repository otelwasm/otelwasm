package imports

import "github.com/musaprg/otelwasm/guest/internal/mem"

//go:wasmimport opentelemetry.io/wasm currentTelemetry
func currentTelemetry(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm setResultTraces
func setResultTraces(ptr, size uint32)
