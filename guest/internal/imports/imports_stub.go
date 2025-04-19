//go:build !wasm

package imports

import "github.com/musaprg/otelwasm/guest/internal/mem"

// This file is used to stub out the imports for running tests.

func currentTraces(ptr uint32, limit mem.BufLimit) (len uint32) { return }

func currentMetrics(ptr uint32, limit mem.BufLimit) (len uint32) { return }

func currentLogs(ptr uint32, limit mem.BufLimit) (len uint32) { return }

func setResultTraces(ptr, size uint32) { return }

func setResultMetrics(ptr, size uint32) { return }

func setResultLogs(ptr, size uint32) { return }

func setResultStatusReason(ptr, size uint32) { return }

func getShutdownRequested() uint32 { return 0 }
