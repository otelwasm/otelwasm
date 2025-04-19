package wasmreceiver

import (
	"testing"

	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestProcessMetricsWithNopProcessor(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()
	ctx, wasmProc, err := newWasmReceiver(ctx, cfg, consumertest.NewNop())
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
