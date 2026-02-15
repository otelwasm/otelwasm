package wasmplugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestPushModelConsumeTraces(t *testing.T) {
	t.Run("returns error when alloc export is missing", func(t *testing.T) {
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: "consume_traces", typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{"consume_traces"})

		_, err := p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil || !strings.Contains(err.Error(), "alloc") {
			t.Fatalf("expected alloc error, got: %v", err)
		}
	})

	t.Run("returns error when alloc returns null pointer", func(t *testing.T) {
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: "alloc", typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(0)},
			{name: "consume_traces", typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{"consume_traces"})

		_, err := p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil || !strings.Contains(err.Error(), "alloc returned null") {
			t.Fatalf("expected null alloc error, got: %v", err)
		}
	})

	t.Run("returns error when guest memory write fails", func(t *testing.T) {
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: "alloc", typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(65535)},
			{name: "consume_traces", typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{"consume_traces"})

		_, err := p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil || !strings.Contains(err.Error(), "failed to write traces payload") {
			t.Fatalf("expected memory write error, got: %v", err)
		}
	})

	t.Run("returns status error from consume_traces", func(t *testing.T) {
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: "alloc", typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(16)},
			{name: "consume_traces", typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(1)},
		}), []string{"consume_traces"})

		_, err := p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil || !strings.Contains(err.Error(), "ERROR") {
			t.Fatalf("expected status error, got: %v", err)
		}
	})

	t.Run("returns original traces when consume succeeds without result", func(t *testing.T) {
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: "alloc", typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(16)},
			{name: "consume_traces", typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{"consume_traces"})

		in := newNonEmptyTraces()
		out, err := p.ConsumeTraces(context.Background(), in)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if out.SpanCount() != in.SpanCount() {
			t.Fatalf("expected unchanged traces span count %d, got %d", in.SpanCount(), out.SpanCount())
		}
	})
}

func TestNewWasmPlugin(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects modules without abi_version_v1 export", func(t *testing.T) {
		modPath := writeTempModule(t, buildTestModule(true, []wasmFunctionSpec{
			{name: "_initialize", typeIndex: wasmTypeFunc0To0},
			{name: "get_supported_telemetry", typeIndex: wasmTypeFunc0ToI32, returnValue: uint32Ptr(0)},
		}))

		cfg := &Config{
			Path:          modPath,
			RuntimeConfig: RuntimeConfig{Mode: RuntimeModeInterpreter},
		}

		p, err := NewWasmPlugin(ctx, cfg, []string{"consume_traces"})
		if p != nil {
			t.Fatal("expected nil plugin on ABI marker failure")
		}
		if !errors.Is(err, ErrABIVersionMarkerNotExported) {
			t.Fatalf("expected ErrABIVersionMarkerNotExported, got: %v", err)
		}
		if err == nil || !strings.Contains(err.Error(), abiVersionV1MarkerExport) {
			t.Fatalf("expected error mentioning %q, got: %v", abiVersionV1MarkerExport, err)
		}
	})
}

func TestRequiresABIV1(t *testing.T) {
	if !requiresABIV1([]string{"consume_traces"}) {
		t.Fatal("consume_traces should require ABI v1")
	}
	if !requiresABIV1([]string{"start", "shutdown"}) {
		t.Fatal("start/shutdown should require ABI v1")
	}
	if !requiresABIV1([]string{"startTracesReceiver"}) {
		t.Fatal("startTracesReceiver should require ABI v1")
	}
	if !requiresABIV1([]string{"startMetricsReceiver"}) {
		t.Fatal("startMetricsReceiver should require ABI v1")
	}
	if !requiresABIV1([]string{"startLogsReceiver"}) {
		t.Fatal("startLogsReceiver should require ABI v1")
	}
	if requiresABIV1([]string{"processTraces"}) {
		t.Fatal("legacy function should not require ABI v1")
	}
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
