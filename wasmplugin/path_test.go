package wasmplugin

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadWasmModuleLoadsLocalPath(t *testing.T) {
	want := []byte("\x00asm fake module")
	path := filepath.Join(t.TempDir(), "main.wasm")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	got, err := readWasmModule(t.Context(), path)
	if err != nil {
		t.Fatalf("failed to read local path: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("local path returned different wasm bytes")
	}
}

func TestReadWasmModuleRejectsUnsupportedScheme(t *testing.T) {
	_, err := readWasmModule(t.Context(), "file:///tmp/main.wasm")
	if err == nil {
		t.Fatal("expected file URL to fail")
	}
	if !strings.Contains(err.Error(), "unsupported path scheme: file") {
		t.Fatalf("expected unsupported scheme error, got %v", err)
	}
}

func TestReadWasmModuleRejectsInvalidOCIReference(t *testing.T) {
	_, err := readWasmModule(t.Context(), "oci://not a valid ref")
	if err == nil {
		t.Fatal("expected invalid OCI reference to fail")
	}
}
