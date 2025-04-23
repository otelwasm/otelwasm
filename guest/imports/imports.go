//go:build wasm

package imports

//go:wasmimport opentelemetry.io/wasm getPluginConfig
func getPluginConfig(ptr, size uint32) (len uint32)

//go:wasmimport opentelemetry.io/wasm setResultTraces
func setResultTraces(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm setResultMetrics
func setResultMetrics(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm setResultLogs
func setResultLogs(ptr, size uint32)
