package wasmplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	wasmTypeFunc0To0 = iota
	wasmTypeFunc0ToI32
	wasmTypeFuncI32ToI32
	wasmTypeFuncI32I32ToI32
)

type wasmFunctionSpec struct {
	name        string
	typeIndex   byte
	returnValue *uint32
}

func newPushTestPlugin(t *testing.T, moduleBytes []byte, exportedFunctions []string) *WasmPlugin {
	t.Helper()

	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	t.Cleanup(func() {
		if err := r.Close(ctx); err != nil {
			t.Fatalf("failed to close runtime: %v", err)
		}
	})

	mod, err := r.Instantiate(ctx, moduleBytes)
	if err != nil {
		t.Fatalf("failed to instantiate module: %v", err)
	}

	fnMap := make(map[string]api.Function, len(exportedFunctions))
	for _, name := range exportedFunctions {
		fn := mod.ExportedFunction(name)
		if fn == nil {
			t.Fatalf("exported function %q was not found", name)
		}
		fnMap[name] = fn
	}

	return &WasmPlugin{
		Runtime:           r,
		Module:            mod,
		ExportedFunctions: fnMap,
	}
}

func writeTempModule(t *testing.T, module []byte) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.wasm")
	if err := os.WriteFile(path, module, 0o600); err != nil {
		t.Fatalf("failed to write test module: %v", err)
	}
	return path
}

func buildTestModule(exportMemory bool, functions []wasmFunctionSpec) []byte {
	module := []byte{
		0x00, 0x61, 0x73, 0x6d, // magic
		0x01, 0x00, 0x00, 0x00, // version
	}

	appendSection := func(sectionID byte, payload []byte) {
		module = append(module, sectionID)
		module = append(module, encodeULEB128Test(uint32(len(payload)))...)
		module = append(module, payload...)
	}

	// Type section:
	// 0: () -> ()
	// 1: () -> i32
	// 2: (i32) -> i32
	// 3: (i32, i32) -> i32
	appendSection(0x01, []byte{
		0x04,             // 4 types
		0x60, 0x00, 0x00, // () -> ()
		0x60, 0x00, 0x01, 0x7f, // () -> i32
		0x60, 0x01, 0x7f, 0x01, 0x7f, // (i32) -> i32
		0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f, // (i32, i32) -> i32
	})

	// Function section
	funcPayload := append([]byte{}, encodeULEB128Test(uint32(len(functions)))...)
	for _, fn := range functions {
		funcPayload = append(funcPayload, fn.typeIndex)
	}
	appendSection(0x03, funcPayload)

	if exportMemory {
		// Memory section: one memory, min 1 page.
		appendSection(0x05, []byte{
			0x01, // 1 memory
			0x00, // only min limit
			0x01, // min 1 page
		})
	}

	// Export section
	exportCount := len(functions)
	if exportMemory {
		exportCount++
	}
	exportPayload := append([]byte{}, encodeULEB128Test(uint32(exportCount))...)
	if exportMemory {
		exportPayload = append(exportPayload, encodeULEB128Test(uint32(len(guestExportMemory)))...)
		exportPayload = append(exportPayload, guestExportMemory...)
		exportPayload = append(exportPayload, 0x02, 0x00) // memory index 0
	}
	for i, fn := range functions {
		exportPayload = append(exportPayload, encodeULEB128Test(uint32(len(fn.name)))...)
		exportPayload = append(exportPayload, fn.name...)
		exportPayload = append(exportPayload, 0x00) // export kind: func
		exportPayload = append(exportPayload, encodeULEB128Test(uint32(i))...)
	}
	appendSection(0x07, exportPayload)

	// Code section
	codePayload := append([]byte{}, encodeULEB128Test(uint32(len(functions)))...)
	for _, fn := range functions {
		body := wasmFunctionBody(fn.returnValue)
		codePayload = append(codePayload, encodeULEB128Test(uint32(len(body)))...)
		codePayload = append(codePayload, body...)
	}
	appendSection(0x0a, codePayload)

	return module
}

func wasmFunctionBody(returnValue *uint32) []byte {
	if returnValue == nil {
		return []byte{
			0x00, // local decl count
			0x0b, // end
		}
	}

	body := []byte{
		0x00, // local decl count
		0x41, // i32.const
	}
	body = append(body, encodeULEB128Test(*returnValue)...)
	body = append(body, 0x0b) // end
	return body
}

func encodeULEB128Test(v uint32) []byte {
	var out []byte
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		out = append(out, b)
		if v == 0 {
			return out
		}
	}
}

func uint32Ptr(v uint32) *uint32 {
	return &v
}

func newNonEmptyTraces() ptrace.Traces {
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("span")
	return td
}
