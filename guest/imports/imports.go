//go:build wasm

package imports

//go:wasmimport opentelemetry.io/wasm otelwasm_get_plugin_config
func getPluginConfig(ptr, size uint32) (len uint32)

//go:wasmimport opentelemetry.io/wasm otelwasm_set_result_traces
func setResultTraces(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm otelwasm_set_result_metrics
func setResultMetrics(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm otelwasm_set_result_logs
func setResultLogs(ptr, size uint32)
