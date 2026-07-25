package wasmprocessor

import (
	"fmt"
	"testing"

	"github.com/otelwasm/otelwasm/wasmplugin"
)

// handleGuestFlavors are the handle-ABI guests implementing the identical
// walk + attribute upsert workload in different languages.
//
//   - go: Go 1.24 c-shared via the guest/pdatahandle SDK (~27 MB)
//   - tinygo: TinyGo, direct wasmimport declarations (~23 KB)
//   - rust: Rust wasm32-wasip1 cdylib, direct extern imports (~650 B)
var handleGuestFlavors = []struct {
	flavor string
	guest  string
}{
	{"go", "logs_add_attribute_handle"},
	{"tinygo", "logs_add_attribute_handle_tinygo"},
	{"rust", "logs_add_attribute_handle_rust"},
}

// BenchmarkHandleLogsGuestFlavors compares guest languages on the wazero
// runtime for the handle-ABI logs workload of BenchmarkHandleVsProtoLogs.
func BenchmarkHandleLogsGuestFlavors(b *testing.B) {
	for _, mode := range []wasmplugin.RuntimeMode{wasmplugin.RuntimeModeInterpreter, wasmplugin.RuntimeModeCompiled} {
		for _, g := range handleGuestFlavors {
			for _, numRecords := range []int{1, 100, 1000} {
				name := fmt.Sprintf("%s/%s/records=%d", mode, g.flavor, numRecords)
				b.Run(name, func(b *testing.B) {
					benchmarkWasmLogs(b, g.guest, mode, numRecords)
				})
			}
		}
	}
}
