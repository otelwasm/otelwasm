//go:build !wasm

package imports

// This file is used to stub out the imports for running tests.

func getPluginConfig(ptr, size uint32) (len uint32) { return }

func setResultTraces(ptr, size uint32) { return }

func setResultMetrics(ptr, size uint32) { return }

func setResultLogs(ptr, size uint32) { return }
