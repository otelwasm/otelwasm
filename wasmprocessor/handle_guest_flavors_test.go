package wasmprocessor

import (
	"testing"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

// TestHandleLogsAddAttributeGuestFlavors verifies that the handle-ABI guests
// written in other languages (TinyGo, Rust) behave the same as the Go one.
func TestHandleLogsAddAttributeGuestFlavors(t *testing.T) {
	for _, name := range []string{"logs_add_attribute_handle_tinygo", "logs_add_attribute_handle_rust"} {
		t.Run(name, func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.Path = "testdata/" + name + "/main.wasm"
			ctx := t.Context()

			sink := new(consumertest.LogsSink)
			settings := processortest.NewNopSettings(typeStr)
			lp, err := factory.CreateLogs(ctx, settings, cfg, sink)
			if err != nil {
				t.Fatalf("failed to create logs processor: %v", err)
			}
			if err := lp.Start(ctx, componenttest.NewNopHost()); err != nil {
				t.Fatalf("failed to start processor: %v", err)
			}
			defer func() {
				if err := lp.Shutdown(ctx); err != nil {
					t.Errorf("failed to shutdown processor: %v", err)
				}
			}()

			ld := generateLogsWithRecords(3)
			if err := lp.ConsumeLogs(ctx, ld); err != nil {
				t.Fatalf("failed to consume logs: %v", err)
			}

			got := sink.AllLogs()
			if len(got) != 1 {
				t.Fatalf("expected 1 logs batch, got %d", len(got))
			}
			records := got[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
			if records.Len() != 3 {
				t.Fatalf("expected 3 log records, got %d", records.Len())
			}
			for i := 0; i < records.Len(); i++ {
				lr := records.At(i)
				v, ok := lr.Attributes().Get("example_key")
				if !ok {
					t.Fatalf("record %d: attribute example_key not found", i)
				}
				if v.Str() != "example_value" {
					t.Fatalf("record %d: expected example_value, got %q", i, v.Str())
				}
			}
		})
	}
}
