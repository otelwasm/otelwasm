package wasmplugin

import (
	"bytes"
	"testing"
)

// fakeMemory is an in-memory implementation of Memory for tests.
type fakeMemory struct {
	buf []byte
}

func (m *fakeMemory) Read(offset, byteCount uint32) ([]byte, bool) {
	if uint64(offset)+uint64(byteCount) > uint64(len(m.buf)) {
		return nil, false
	}
	return m.buf[offset : offset+byteCount], true
}

func (m *fakeMemory) Write(offset uint32, v []byte) bool {
	if uint64(offset)+uint64(len(v)) > uint64(len(m.buf)) {
		return false
	}
	copy(m.buf[offset:], v)
	return true
}

func TestWriteBytesIfUnderLimit(t *testing.T) {
	data := []byte("hello, wasm")

	t.Run("fits within limit", func(t *testing.T) {
		mem := &fakeMemory{buf: make([]byte, 64)}
		size := writeBytesIfUnderLimit(mem, data, 0, 32)
		if size != uint32(len(data)) {
			t.Errorf("expected size %d, got %d", len(data), size)
		}
		if got, _ := mem.Read(0, size); !bytes.Equal(got, data) {
			t.Errorf("expected %q written to memory, got %q", data, got)
		}
	})

	t.Run("exceeds limit returns required size without writing", func(t *testing.T) {
		// The guest grows its buffer to the returned size and calls again,
		// so the host must report the required size instead of 0.
		mem := &fakeMemory{buf: make([]byte, 64)}
		size := writeBytesIfUnderLimit(mem, data, 0, uint32(len(data)-1))
		if size != uint32(len(data)) {
			t.Errorf("expected required size %d, got %d", len(data), size)
		}
		if got, _ := mem.Read(0, uint32(len(data))); !bytes.Equal(got, make([]byte, len(data))) {
			t.Errorf("expected memory to be untouched, got %q", got)
		}
	})

	t.Run("write out of range returns 0", func(t *testing.T) {
		mem := &fakeMemory{buf: make([]byte, 4)}
		if size := writeBytesIfUnderLimit(mem, data, 0, 32); size != 0 {
			t.Errorf("expected 0 for out-of-range write, got %d", size)
		}
	})
}
