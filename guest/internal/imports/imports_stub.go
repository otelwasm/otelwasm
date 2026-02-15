//go:build !wasm

package imports

import "github.com/otelwasm/otelwasm/guest/internal/mem"

// This file is used to stub out the imports for running tests.

func currentTraces(ptr uint32, limit mem.BufLimit) (len uint32) { return }

func currentMetrics(ptr uint32, limit mem.BufLimit) (len uint32) { return }

func currentLogs(ptr uint32, limit mem.BufLimit) (len uint32) { return }

func setStatusReasonHost(ptr, size uint32) { return }

func getShutdownRequested() uint32 { return 0 }
