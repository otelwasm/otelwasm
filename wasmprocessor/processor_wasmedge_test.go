//go:build wasmedge

package wasmprocessor

import (
	"os"
	"testing"

	"github.com/otelwasm/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

func newWasmEdgeConfig(path string) *Config {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = path
	cfg.RuntimeConfig.Name = wasmplugin.RuntimeNameWasmEdge
	cfg.RuntimeConfig.Mode = ""
	return cfg
}

func TestWasmEdgeProcessTraces(t *testing.T) {
	factory := NewFactory()
	cfg := newWasmEdgeConfig("testdata/nop/main.wasm")
	ctx := t.Context()

	sink := new(consumertest.TracesSink)
	settings := processortest.NewNopSettings(typeStr)
	tp, err := factory.CreateTraces(ctx, settings, cfg, sink)
	if err != nil {
		t.Fatalf("failed to create traces processor: %v", err)
	}

	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "wasmedge-test")
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	span.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))

	if err := tp.ConsumeTraces(ctx, traces); err != nil {
		t.Fatalf("failed to consume traces: %v", err)
	}

	got := sink.AllTraces()
	if len(got) != 1 {
		t.Fatalf("expected 1 traces batch, got %d", len(got))
	}
	if got[0].SpanCount() != 1 {
		t.Errorf("expected 1 span, got %d", got[0].SpanCount())
	}
	gotSpan := got[0].ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	if gotSpan.Name() != "test-span" {
		t.Errorf("expected span name %q, got %q", "test-span", gotSpan.Name())
	}

	if err := tp.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown processor: %v", err)
	}
}

// TestWasmEdgeHandleLogsAddAttribute verifies the handle-based pdata ABI on
// the WasmEdge runtime, using the same guests and expectations as
// TestHandleLogsAddAttribute.
func TestWasmEdgeHandleLogsAddAttribute(t *testing.T) {
	for _, name := range []string{"logs_add_attribute_handle", "logs_add_attribute_proto"} {
		t.Run(name, func(t *testing.T) {
			factory := NewFactory()
			cfg := newWasmEdgeConfig("testdata/" + name + "/main.wasm")
			ctx := t.Context()

			sink := new(consumertest.LogsSink)
			settings := processortest.NewNopSettings(typeStr)
			lp, err := factory.CreateLogs(ctx, settings, cfg, sink)
			if err != nil {
				t.Fatalf("failed to create logs processor: %v", err)
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

// TestWasmEdgeAOTCompiledMode exercises runtime.mode "compiled", which
// AOT-compiles the guest at startup. AOT compilation takes minutes for the
// Go-based guests, so this test only runs when OTELWASM_TEST_AOT is set.
func TestWasmEdgeAOTCompiledMode(t *testing.T) {
	if os.Getenv("OTELWASM_TEST_AOT") == "" {
		t.Skip("set OTELWASM_TEST_AOT=1 to run the slow AOT compilation test")
	}

	factory := NewFactory()
	cfg := newWasmEdgeConfig("testdata/nop/main.wasm")
	cfg.RuntimeConfig.Mode = wasmplugin.RuntimeModeCompiled
	ctx := t.Context()

	sink := new(consumertest.TracesSink)
	settings := processortest.NewNopSettings(typeStr)
	tp, err := factory.CreateTraces(ctx, settings, cfg, sink)
	if err != nil {
		t.Fatalf("failed to create traces processor: %v", err)
	}

	traces := ptrace.NewTraces()
	span := traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("aot-span")

	if err := tp.ConsumeTraces(ctx, traces); err != nil {
		t.Fatalf("failed to consume traces: %v", err)
	}
	if len(sink.AllTraces()) != 1 {
		t.Fatalf("expected 1 traces batch, got %d", len(sink.AllTraces()))
	}

	if err := tp.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown processor: %v", err)
	}
}

func TestWasmEdgeProcessTracesWithPluginConfig(t *testing.T) {
	factory := NewFactory()
	cfg := newWasmEdgeConfig("testdata/add_new_attribute/main.wasm")
	cfg.PluginConfig = wasmplugin.PluginConfig{
		"attribute_name":  "wasmedge.key",
		"attribute_value": "wasmedge-value",
	}
	ctx := t.Context()

	sink := new(consumertest.TracesSink)
	settings := processortest.NewNopSettings(typeStr)
	tp, err := factory.CreateTraces(ctx, settings, cfg, sink)
	if err != nil {
		t.Fatalf("failed to create traces processor: %v", err)
	}

	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("test-span")

	if err := tp.ConsumeTraces(ctx, traces); err != nil {
		t.Fatalf("failed to consume traces: %v", err)
	}

	got := sink.AllTraces()
	if len(got) != 1 {
		t.Fatalf("expected 1 traces batch, got %d", len(got))
	}
	gotSpan := got[0].ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
	v, ok := gotSpan.Attributes().Get("wasmedge.key")
	if !ok {
		t.Fatalf("expected attribute %q to be added by the plugin", "wasmedge.key")
	}
	if v.Str() != "wasmedge-value" {
		t.Errorf("expected attribute value %q, got %q", "wasmedge-value", v.Str())
	}

	if err := tp.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown processor: %v", err)
	}
}
