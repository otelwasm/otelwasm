//go:build wasm

package imports

//go:wasmimport opentelemetry.io/wasm otelwasm_set_status_reason
func setStatusReasonHost(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm otelwasm_get_shutdown_requested
func getShutdownRequested() uint32
