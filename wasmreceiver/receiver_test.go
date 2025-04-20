package wasmreceiver

import (
	"testing"

	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestProcessMetricsWithNopReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()
	ctx, wasmProc, err := newMetricsWasmReceiver(ctx, cfg, consumertest.NewNop())
	if err != nil {
		t.Fatalf("failed to create wasm receiver: %v", err)
	}

	// Start the metrics receiver
	err = wasmProc.Start(ctx, nil)
	if err != nil {
		t.Fatalf("failed to start wasm receiver: %v", err)
	}

	// Stop the metrics receiver
	err = wasmProc.Shutdown(ctx)
	if err != nil {
		t.Fatalf("failed to stop wasm receiver: %v", err)
	}
}

func TestProcessLogsWithNopReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()
	ctx, wasmProc, err := newLogsWasmReceiver(ctx, cfg, consumertest.NewNop())
	if err != nil {
		t.Fatalf("failed to create wasm receiver: %v", err)
	}

	// Start the metrics receiver
	err = wasmProc.Start(ctx, nil)
	if err != nil {
		t.Fatalf("failed to start wasm receiver: %v", err)
	}

	// Stop the metrics receiver
	err = wasmProc.Shutdown(ctx)
	if err != nil {
		t.Fatalf("failed to stop wasm receiver: %v", err)
	}
}

func TestProcessTracesWithNopReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()
	ctx, wasmProc, err := newTracesWasmReceiver(ctx, cfg, consumertest.NewNop())
	if err != nil {
		t.Fatalf("failed to create wasm receiver: %v", err)
	}

	// Start the metrics receiver
	err = wasmProc.Start(ctx, nil)
	if err != nil {
		t.Fatalf("failed to start wasm receiver: %v", err)
	}

	// Stop the metrics receiver
	err = wasmProc.Shutdown(ctx)
	if err != nil {
		t.Fatalf("failed to stop wasm receiver: %v", err)
	}
}
