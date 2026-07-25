//go:build !wasmedge

package wasmplugin

import (
	"context"
	"errors"
)

// ErrWasmEdgeNotEnabled is returned when the wasmedge runtime is requested
// but the binary was built without WasmEdge support.
var ErrWasmEdgeNotEnabled = errors.New("wasmedge runtime support is not compiled in; rebuild with -tags wasmedge and the WasmEdge shared library installed")

// newWasmEdgeRuntime is a stub used when the binary is built without the
// "wasmedge" build tag. The real implementation lives in runtime_wasmedge.go.
func newWasmEdgeRuntime(ctx context.Context, guestPath string, guestBin []byte, rc RuntimeConfig) (guestRuntime, error) {
	return nil, ErrWasmEdgeNotEnabled
}
