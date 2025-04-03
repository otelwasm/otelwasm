package slices

import (
	"context"

	"github.com/tetratelabs/wazero/api"
)

func CopyBytesToWasm(ctx context.Context, m api.Module, b []byte) uint64 {
	malloc := m.ExportedFunction("malloc")
	sLen := len(b)
	res, _ := malloc.Call(ctx, uint64(sLen))
	resPtr := uint32(res[0])

	_ = m.Memory().Write(resPtr, b)

	return (uint64(resPtr) << uint64(32)) | uint64(len(b))
}
