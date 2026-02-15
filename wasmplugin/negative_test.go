package wasmplugin

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestABIV1BoundaryNegativeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects modules without abi_version_v1 export", func(t *testing.T) {
		modPath := writeTempModule(t, buildTestModule(true, []wasmFunctionSpec{
			{name: "_initialize", typeIndex: wasmTypeFunc0To0},
			{name: allocFunction, typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(16)},
			{name: consumeTracesFunction, typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
			{name: getSupportedTelemetry, typeIndex: wasmTypeFunc0ToI32, returnValue: uint32Ptr(uint32(telemetryTypeTraces))},
		}))

		p, err := NewWasmPlugin(ctx, &Config{
			Path:          modPath,
			RuntimeConfig: RuntimeConfig{Mode: RuntimeModeInterpreter},
		}, []string{consumeTracesFunction})
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

	t.Run("rejects modules missing required consume_traces export", func(t *testing.T) {
		modPath := writeTempModule(t, buildTestModule(true, []wasmFunctionSpec{
			{name: "_initialize", typeIndex: wasmTypeFunc0To0},
			{name: abiVersionV1MarkerExport, typeIndex: wasmTypeFunc0To0},
			{name: allocFunction, typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(16)},
			{name: getSupportedTelemetry, typeIndex: wasmTypeFunc0ToI32, returnValue: uint32Ptr(uint32(telemetryTypeTraces))},
		}))

		p, err := NewWasmPlugin(ctx, &Config{
			Path:          modPath,
			RuntimeConfig: RuntimeConfig{Mode: RuntimeModeInterpreter},
		}, []string{consumeTracesFunction})
		if p != nil {
			t.Fatal("expected nil plugin when required function is missing")
		}
		if !errors.Is(err, ErrRequiredFunctionNotExported) {
			t.Fatalf("expected ErrRequiredFunctionNotExported, got: %v", err)
		}
		if err == nil || !strings.Contains(err.Error(), consumeTracesFunction) {
			t.Fatalf("expected error mentioning %q, got: %v", consumeTracesFunction, err)
		}
	})

	t.Run("consume_traces returns alloc call failure", func(t *testing.T) {
		// Export alloc with the wrong signature so the host call fails.
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: allocFunction, typeIndex: wasmTypeFunc0To0},
			{name: consumeTracesFunction, typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{consumeTracesFunction})

		_, err := p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil || !strings.Contains(err.Error(), "failed to call alloc") {
			t.Fatalf("expected alloc call failure, got: %v", err)
		}
	})

	t.Run("propagates status reason returned by consume_traces", func(t *testing.T) {
		modPath := writeTempModule(t, buildStatusReasonConsumeTracesModule("guest refused traces"))
		p, err := NewWasmPlugin(ctx, &Config{
			Path:          modPath,
			RuntimeConfig: RuntimeConfig{Mode: RuntimeModeInterpreter},
		}, []string{consumeTracesFunction})
		if err != nil {
			t.Fatalf("expected plugin creation to succeed, got: %v", err)
		}
		t.Cleanup(func() {
			if closeErr := p.Shutdown(context.Background()); closeErr != nil {
				t.Fatalf("failed to shutdown plugin: %v", closeErr)
			}
		})

		_, err = p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil {
			t.Fatal("expected consume_traces error, got nil")
		}
		if !strings.Contains(err.Error(), "ERROR") {
			t.Fatalf("expected status string in error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "guest refused traces") {
			t.Fatalf("expected propagated status reason, got: %v", err)
		}
	})
}

func buildStatusReasonConsumeTracesModule(reason string) []byte {
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
	// 0: (i32, i32) -> ()      [set_status_reason import]
	// 1: (i32) -> i32          [alloc]
	// 2: (i32, i32) -> i32     [consume_traces]
	// 3: () -> ()              [abi marker / _initialize]
	// 4: () -> i32             [get_supported_telemetry]
	appendSection(0x01, []byte{
		0x05, // 5 types
		0x60, 0x02, 0x7f, 0x7f, 0x00,
		0x60, 0x01, 0x7f, 0x01, 0x7f,
		0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f,
		0x60, 0x00, 0x00,
		0x60, 0x00, 0x01, 0x7f,
	})

	// Import section: import opentelemetry.io/wasm.set_status_reason as function index 0.
	importPayload := []byte{0x01}
	importPayload = append(importPayload, encodeULEB128Test(uint32(len(otelWasm)))...)
	importPayload = append(importPayload, otelWasm...)
	importPayload = append(importPayload, encodeULEB128Test(uint32(len(setStatusReason)))...)
	importPayload = append(importPayload, setStatusReason...)
	importPayload = append(importPayload, 0x00, 0x00) // kind=func, type index 0
	appendSection(0x02, importPayload)

	// Function section: 5 local functions (indices 1..5).
	appendSection(0x03, []byte{
		0x05, // 5 functions
		0x01, // alloc
		0x02, // consume_traces
		0x03, // abi_version_v1
		0x04, // get_supported_telemetry
		0x03, // _initialize
	})

	// Memory section: one memory, min 1 page.
	appendSection(0x05, []byte{
		0x01, // 1 memory
		0x00, // min only
		0x01, // min pages
	})

	// Export section.
	// Function indices include imports, so local functions start at 1.
	exportPayload := []byte{0x06}
	exportPayload = append(exportPayload, encodeULEB128Test(uint32(len(guestExportMemory)))...)
	exportPayload = append(exportPayload, guestExportMemory...)
	exportPayload = append(exportPayload, 0x02, 0x00) // memory index 0
	exportPayload = appendExportedFunc(exportPayload, allocFunction, 1)
	exportPayload = appendExportedFunc(exportPayload, consumeTracesFunction, 2)
	exportPayload = appendExportedFunc(exportPayload, abiVersionV1MarkerExport, 3)
	exportPayload = appendExportedFunc(exportPayload, getSupportedTelemetry, 4)
	exportPayload = appendExportedFunc(exportPayload, "_initialize", 5)
	appendSection(0x07, exportPayload)

	// Code section.
	// alloc(size) -> i32.const 4096
	allocBody := []byte{0x00, 0x41}
	allocBody = append(allocBody, encodeULEB128Test(4096)...)
	allocBody = append(allocBody, 0x0b)

	// consume_traces(data_ptr, data_size):
	//   call set_status_reason(reason_offset=32, reason_len=len(reason))
	//   return ERROR(1)
	consumeBody := []byte{
		0x00,                   // local decl count
		0x41, 0x20,             // i32.const 32
		0x41, byte(len(reason)), // i32.const reason_len (len(reason) < 128 in tests)
		0x10, 0x00,             // call function index 0 (imported set_status_reason)
		0x41, 0x01,             // i32.const 1 (ERROR)
		0x0b, // end
	}

	abiMarkerBody := []byte{0x00, 0x0b}
	getSupportedBody := []byte{
		0x00,
		0x41, byte(telemetryTypeTraces),
		0x0b,
	}
	initializeBody := []byte{0x00, 0x0b}

	codePayload := []byte{0x05}
	for _, body := range [][]byte{
		allocBody,
		consumeBody,
		abiMarkerBody,
		getSupportedBody,
		initializeBody,
	} {
		codePayload = append(codePayload, encodeULEB128Test(uint32(len(body)))...)
		codePayload = append(codePayload, body...)
	}
	appendSection(0x0a, codePayload)

	// Data section: write the status reason into guest memory at offset 32.
	dataPayload := []byte{
		0x01,       // 1 segment
		0x00,       // active segment for memory index 0
		0x41, 0x20, // i32.const 32
		0x0b, // end
	}
	dataPayload = append(dataPayload, encodeULEB128Test(uint32(len(reason)))...)
	dataPayload = append(dataPayload, reason...)
	appendSection(0x0b, dataPayload)

	return module
}

func appendExportedFunc(payload []byte, name string, funcIndex byte) []byte {
	payload = append(payload, encodeULEB128Test(uint32(len(name)))...)
	payload = append(payload, name...)
	payload = append(payload, 0x00, funcIndex) // kind=func, function index
	return payload
}
