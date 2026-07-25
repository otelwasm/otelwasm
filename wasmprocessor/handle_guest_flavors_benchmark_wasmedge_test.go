//go:build wasmedge

package wasmprocessor

import (
	"fmt"
	"os"
	"testing"
)

// BenchmarkHandleLogsGuestFlavorsWasmEdge compares guest languages on the
// WasmEdge runtime for the handle-ABI logs workload. The "aot" executions
// use guests precompiled with the WasmEdge AOT compiler.
func BenchmarkHandleLogsGuestFlavorsWasmEdge(b *testing.B) {
	for _, exec := range []string{"interpreter", "aot"} {
		for _, g := range handleGuestFlavors {
			for _, numRecords := range []int{1, 100, 1000} {
				name := fmt.Sprintf("%s/%s/records=%d", exec, g.flavor, numRecords)
				b.Run(name, func(b *testing.B) {
					file := "main.wasm"
					if exec == "aot" {
						file = "main-aot.wasm"
					}
					path := "testdata/" + g.guest + "/" + file
					if _, err := os.Stat(path); err != nil {
						b.Skipf("guest not found at %s; generate it with: wasmedge compile --optimize 2", path)
					}
					benchmarkWasmEdgeLogs(b, path, numRecords)
				})
			}
		}
	}
}
