//go:build wasm

package imports

import "github.com/otelwasm/otelwasm/guest/internal/mem"

//go:wasmimport opentelemetry.io/wasm currentTraces
func currentTraces(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm currentMetrics
func currentMetrics(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm currentLogs
func currentLogs(ptr uint32, limit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm set_status_reason
func setStatusReasonHost(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm get_shutdown_requested
func getShutdownRequested() uint32
