package wasmplugin

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestPushModelConsumeTraces(t *testing.T) {
	t.Run("returns error when memory allocate export is missing", func(t *testing.T) {
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: consumeTracesFunction, typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{consumeTracesFunction})

		_, err := p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil || !strings.Contains(err.Error(), memoryAllocateFunction) {
			t.Fatalf("expected missing %q error, got: %v", memoryAllocateFunction, err)
		}
	})

	t.Run("returns error when memory allocate returns null pointer", func(t *testing.T) {
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: memoryAllocateFunction, typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(0)},
			{name: consumeTracesFunction, typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{consumeTracesFunction})

		_, err := p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil || !strings.Contains(err.Error(), memoryAllocateFunction+" returned null") {
			t.Fatalf("expected null %q error, got: %v", memoryAllocateFunction, err)
		}
	})

	t.Run("returns error when guest memory write fails", func(t *testing.T) {
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: memoryAllocateFunction, typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(65535)},
			{name: consumeTracesFunction, typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{consumeTracesFunction})

		_, err := p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil || !strings.Contains(err.Error(), "failed to write traces payload") {
			t.Fatalf("expected memory write error, got: %v", err)
		}
	})

	t.Run("returns status error from otelwasm_consume_traces", func(t *testing.T) {
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: memoryAllocateFunction, typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(16)},
			{name: consumeTracesFunction, typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(1)},
		}), []string{consumeTracesFunction})

		_, err := p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil || !strings.Contains(err.Error(), "ERROR") {
			t.Fatalf("expected status error, got: %v", err)
		}
	})

	t.Run("returns original traces when consume succeeds without result", func(t *testing.T) {
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: memoryAllocateFunction, typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(16)},
			{name: consumeTracesFunction, typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{consumeTracesFunction})

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

	t.Run("rejects modules without otelwasm_abi_version_0_1_0 export", func(t *testing.T) {
		modPath := writeTempModule(t, buildTestModule(true, []wasmFunctionSpec{
			{name: "_initialize", typeIndex: wasmTypeFunc0To0},
			{name: getSupportedTelemetry, typeIndex: wasmTypeFunc0ToI32, returnValue: uint32Ptr(0)},
		}))

		cfg := &Config{
			Path:          modPath,
			RuntimeConfig: RuntimeConfig{Mode: RuntimeModeInterpreter},
		}

		p, err := NewWasmPlugin(ctx, cfg, []string{consumeTracesFunction})
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
	if !requiresABIV1([]string{consumeTracesFunction}) {
		t.Fatal("otelwasm_consume_traces should require ABI v1")
	}
	if !requiresABIV1([]string{"otelwasm_start", "otelwasm_shutdown"}) {
		t.Fatal("otelwasm_start/otelwasm_shutdown should require ABI v1")
	}
	if !requiresABIV1([]string{"otelwasm_start_traces_receiver"}) {
		t.Fatal("otelwasm_start_traces_receiver should require ABI v1")
	}
	if !requiresABIV1([]string{"otelwasm_start_metrics_receiver"}) {
		t.Fatal("otelwasm_start_metrics_receiver should require ABI v1")
	}
	if !requiresABIV1([]string{"otelwasm_start_logs_receiver"}) {
		t.Fatal("otelwasm_start_logs_receiver should require ABI v1")
	}
	if requiresABIV1([]string{"processTraces"}) {
		t.Fatal("legacy function should not require ABI v1")
	}
}
