package mem

import (
	"fmt"
	"unsafe"
)

// pinnedAllocations keeps references to host-allocated buffers until guest code
// takes ownership, preventing the GC from reclaiming them too early.
var pinnedAllocations = map[uint32][]byte{}

// Alloc allocates and pins a byte buffer in guest memory.
//
//go:wasmexport otelwasm_memory_allocate
func Alloc(size uint32) uint32 {
	if size == 0 {
		return 0
	}

	buf := make([]byte, size)
	ptr := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(buf))))
	pinnedAllocations[ptr] = buf
	return ptr
}

// TakeOwnership returns an allocated buffer and unpins it.
func TakeOwnership(ptr uint32, size uint32) []byte {
	if ptr == 0 && size == 0 {
		return nil
	}

	buf, ok := pinnedAllocations[ptr]
	if !ok {
		panic(fmt.Sprintf("TakeOwnership: unknown pointer %d", ptr))
	}

	delete(pinnedAllocations, ptr)
	if size > uint32(len(buf)) {
		panic(fmt.Sprintf("TakeOwnership: size %d exceeds allocation %d", size, len(buf)))
	}

	return buf[:size]
}
