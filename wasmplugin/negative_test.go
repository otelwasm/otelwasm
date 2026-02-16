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
			{name: memoryAllocateFunction, typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(16)},
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

	t.Run("rejects modules missing required otelwasm_consume_traces export", func(t *testing.T) {
		modPath := writeTempModule(t, buildTestModule(true, []wasmFunctionSpec{
			{name: "_initialize", typeIndex: wasmTypeFunc0To0},
			{name: abiVersionV1MarkerExport, typeIndex: wasmTypeFunc0To0},
			{name: memoryAllocateFunction, typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(16)},
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

	t.Run("otelwasm_consume_traces returns memory allocate call failure", func(t *testing.T) {
		// Export memory allocate with the wrong signature so the host call fails.
		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: memoryAllocateFunction, typeIndex: wasmTypeFunc0To0},
			{name: consumeTracesFunction, typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{consumeTracesFunction})

		_, err := p.ConsumeTraces(context.Background(), newNonEmptyTraces())
		if err == nil || !strings.Contains(err.Error(), "failed to call "+memoryAllocateFunction) {
			t.Fatalf("expected %q call failure, got: %v", memoryAllocateFunction, err)
		}
	})

	t.Run("propagates status reason returned by otelwasm_consume_traces", func(t *testing.T) {
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
			t.Fatal("expected otelwasm_consume_traces error, got nil")
		}
		if !strings.Contains(err.Error(), "ERROR") {
			t.Fatalf("expected status string in error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "guest refused traces") {
			t.Fatalf("expected propagated status reason, got: %v", err)
		}
	})
}
