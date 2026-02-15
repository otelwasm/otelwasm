//go:build wasm

package imports

//go:wasmimport opentelemetry.io/wasm set_status_reason
func setStatusReasonHost(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm get_shutdown_requested
func getShutdownRequested() uint32
