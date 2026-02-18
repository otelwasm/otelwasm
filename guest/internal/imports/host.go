package imports

import (
	"runtime"

	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/internal/mem"
)

// StatusToCode returns a WebAssembly compatible result for the input status,
// after sending any reason to the host.
func StatusToCode(s *api.Status) uint32 {
	// Nil status is the same as one with a success code.
	if s == nil || s.Code == api.StatusCodeSuccess {
		return uint32(api.StatusCodeSuccess)
	}

	// WebAssembly Core 2.0 (DRAFT) only includes numeric types. Return the
	// reason using a host function.
	if reason := s.Reason; reason != "" {
		setStatusReason(reason)
	}

	return uint32(s.Code)
}

func setStatusReason(reason string) {
	ptr, size := mem.StringToPtr(reason)
	setStatusReasonHost(ptr, size)
	runtime.KeepAlive(reason) // until ptr is no longer needed.
}

func GetShutdownRequested() bool {
	return getShutdownRequested() != 0
}
