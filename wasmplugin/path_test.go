package wasmplugin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestReadWasmModuleLoadsHTTPPath(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("../wasmprocessor/testdata/nop")))
	defer server.Close()

	got, err := readWasmModule(t.Context(), server.URL+"/main.wasm")
	if err != nil {
		t.Fatalf("failed to read HTTP path: %v", err)
	}
	want, err := os.ReadFile("../wasmprocessor/testdata/nop/main.wasm")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("HTTP path returned different wasm bytes")
	}
}

func TestReadWasmModuleLoadsLocalPath(t *testing.T) {
	got, err := readWasmModule(t.Context(), "../wasmprocessor/testdata/nop/main.wasm")
	if err != nil {
		t.Fatalf("failed to read local path: %v", err)
	}
	want, err := os.ReadFile("../wasmprocessor/testdata/nop/main.wasm")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("local path returned different wasm bytes")
	}
}

func TestReadWasmModuleRejectsFileURLPath(t *testing.T) {
	_, err := readWasmModule(t.Context(), "file:///tmp/main.wasm")
	if err == nil {
		t.Fatal("expected file URL to fail")
	}
	if !strings.Contains(err.Error(), "unsupported path scheme: file") {
		t.Fatalf("expected unsupported scheme error, got %v", err)
	}
}
