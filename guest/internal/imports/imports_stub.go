//go:build !wasm

package imports

// This file is used to stub out the imports for running tests.

func setStatusReasonHost(ptr, size uint32) { return }

func getShutdownRequested() uint32 { return 0 }
