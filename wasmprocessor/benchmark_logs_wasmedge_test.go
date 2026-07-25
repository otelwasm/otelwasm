//go:build wasmedge

package wasmprocessor

import (
	"os"
	"testing"

	"github.com/otelwasm/otelwasm/wasmplugin"
)

// aotWasmPath points to the attributesprocessor guest precompiled with the
// WasmEdge AOT compiler:
//
//	wasmedge compile --optimize 2 testdata/attributesprocessor/main.wasm \
//	    testdata/attributesprocessor/main-aot.wasm
//
// Precompiling keeps the multi-minute AOT compilation out of the benchmark
// setup. Alternatively, runtime.mode "compiled" AOT-compiles at startup.
const aotWasmPath = "testdata/attributesprocessor/main-aot.wasm"

func BenchmarkLogsAttributesProcessorWasmEdgeInterpreter(b *testing.B) {
	benchmarkWasmLogsAttributes(b, "testdata/attributesprocessor/main.wasm",
		wasmplugin.RuntimeNameWasmEdge, wasmplugin.RuntimeModeInterpreter)
}

func BenchmarkLogsAttributesProcessorWasmEdgeAOT(b *testing.B) {
	if _, err := os.Stat(aotWasmPath); err != nil {
		b.Skipf("AOT-compiled guest not found at %s; generate it with: wasmedge compile --optimize 2 testdata/attributesprocessor/main.wasm %s", aotWasmPath, aotWasmPath)
	}

	// The guest is already AOT-compiled, so the runtime just loads it and
	// executes the embedded native code.
	benchmarkWasmLogsAttributes(b, aotWasmPath,
		wasmplugin.RuntimeNameWasmEdge, wasmplugin.RuntimeModeInterpreter)
}
