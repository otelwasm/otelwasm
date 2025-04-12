package imports

//go:wasmimport opentelemetry.io/wasm getPluginConfig
func getPluginConfig(ptr, size uint32) (len uint32)
