package pointer

import (
	"context"
	"log"

	"github.com/tetratelabs/wazero/api"
)

func StringFromWasm(ctx context.Context, m api.Module, sOff uint32, sLen uint32) string {
	buf, ok := m.Memory().Read(sOff, sLen)
	if !ok {
		log.Fatalf("Memory.Read(%d, %d) out of range", sOff, sLen)
	}
	return string(buf)
}
