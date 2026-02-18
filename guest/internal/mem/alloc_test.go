package mem

import (
	"runtime"
	"testing"
)

func resetPinnedAllocations() {
	pinnedAllocations = map[uint32][]byte{}
}

func TestAllocAndTakeOwnership(t *testing.T) {
	resetPinnedAllocations()

	size := uint32(128)
	ptr := Alloc(size)
	if ptr == 0 {
		t.Fatal("Alloc returned null pointer")
	}
	if len(pinnedAllocations) != 1 {
		t.Fatalf("expected 1 pinned allocation, got %d", len(pinnedAllocations))
	}

	buf := TakeOwnership(ptr, size)
	if len(buf) != int(size) {
		t.Fatalf("TakeOwnership returned len=%d, want %d", len(buf), size)
	}
	if len(pinnedAllocations) != 0 {
		t.Fatalf("expected pinned allocations to be empty, got %d", len(pinnedAllocations))
	}
}

func TestAllocZeroSize(t *testing.T) {
	resetPinnedAllocations()

	ptr := Alloc(0)
	if ptr != 0 {
		t.Fatalf("Alloc(0) returned %d, want 0", ptr)
	}
	if len(pinnedAllocations) != 0 {
		t.Fatalf("Alloc(0) should not pin memory, got %d pinned entries", len(pinnedAllocations))
	}

	buf := TakeOwnership(0, 0)
	if buf != nil {
		t.Fatalf("TakeOwnership(0, 0) = %v, want nil", buf)
	}
}

func TestTakeOwnershipPanicsOnUnknownPointer(t *testing.T) {
	resetPinnedAllocations()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unknown pointer")
		}
	}()

	_ = TakeOwnership(42, 1)
}

func TestTakeOwnershipPanicsOnInvalidSize(t *testing.T) {
	resetPinnedAllocations()

	ptr := Alloc(8)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid size")
		}
		if len(pinnedAllocations) != 0 {
			t.Fatalf("expected allocation to be unpinned on panic, got %d", len(pinnedAllocations))
		}
	}()

	_ = TakeOwnership(ptr, 9)
}

func TestAllocStaysPinnedAcrossGC(t *testing.T) {
	resetPinnedAllocations()

	ptr := Alloc(64)
	runtime.GC()
	runtime.GC()

	buf := TakeOwnership(ptr, 64)
	if len(buf) != 64 {
		t.Fatalf("TakeOwnership returned len=%d, want 64", len(buf))
	}
}
