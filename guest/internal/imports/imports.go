//go:build wasm

package imports

import "github.com/otelwasm/otelwasm/guest/internal/mem"

//go:wasmimport opentelemetry.io/wasm currentTraces
func currentTraces(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm currentMetrics
func currentMetrics(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm currentLogs
func currentLogs(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm setResultStatusReason
func setResultStatusReason(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm getShutdownRequested
func getShutdownRequested() uint32
