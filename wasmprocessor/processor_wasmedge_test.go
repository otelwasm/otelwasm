//go:build wasmedge

package wasmprocessor

import (
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
