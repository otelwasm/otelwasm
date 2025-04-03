package slices

import (
	"context"
	"unsafe"

	"github.com/tetratelabs/wazero/api"
)

func CopyStringToWasm(ctx context.Context, m api.Module, s string) uint64 {
	malloc := m.ExportedFunction("malloc")
	sLen := len(s)
	res, _ := malloc.Call(ctx, uint64(sLen))
	resPtr := uint32(res[0])

	// Get byte representation of string without allocation
	buf := unsafe.Slice(unsafe.StringData(s), len(s))

	_ = m.Memory().Write(resPtr, buf)

	return (uint64(resPtr) << uint64(32)) | uint64(len(buf))
}
