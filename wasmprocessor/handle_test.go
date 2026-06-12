package wasmprocessor

import (
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

// generateTracesWithSpans returns traces with a single resource/scope and the
// given number of spans, each with a few attributes.
func generateTracesWithSpans(numSpans int) ptrace.Traces {
	td := ptrace.NewTraces()
	resourceSpans := td.ResourceSpans().AppendEmpty()
	resourceSpans.Resource().Attributes().PutStr("service.name", "benchmark-service")
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
	scopeSpans.Scope().SetName("benchmark-scope")
	start := pcommon.NewTimestampFromTime(time.Unix(1700000000, 0))
	end := pcommon.NewTimestampFromTime(time.Unix(1700000001, 0))
	for i := 0; i < numSpans; i++ {
		span := scopeSpans.Spans().AppendEmpty()
		span.SetName(fmt.Sprintf("span-%d", i))
		span.SetKind(ptrace.SpanKindServer)
		span.SetStartTimestamp(start)
		span.SetEndTimestamp(end)
		attrs := span.Attributes()
		attrs.PutStr("http.method", "GET")
		attrs.PutStr("http.route", "/api/v1/items/:id")
		attrs.PutStr("net.peer.name", "upstream.example.com")
		attrs.PutInt("http.status_code", 200)
		attrs.PutBool("error", false)
	}
	return td
}

func TestHandleAddAttribute(t *testing.T) {
	for _, name := range []string{"add_attribute_handle", "add_attribute_proto"} {
		t.Run(name, func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.Path = "testdata/" + name + "/main.wasm"
			ctx := t.Context()

			sink := new(consumertest.TracesSink)
			settings := processortest.NewNopSettings(typeStr)
			tp, err := factory.CreateTraces(ctx, settings, cfg, sink)
			if err != nil {
				t.Fatalf("failed to create traces processor: %v", err)
			}
			if err := tp.Start(ctx, componenttest.NewNopHost()); err != nil {
				t.Fatalf("failed to start processor: %v", err)
			}
			defer func() {
				if err := tp.Shutdown(ctx); err != nil {
					t.Errorf("failed to shutdown processor: %v", err)
				}
			}()

			td := generateTracesWithSpans(3)
			if err := tp.ConsumeTraces(ctx, td); err != nil {
				t.Fatalf("failed to consume traces: %v", err)
			}

			got := sink.AllTraces()
			if len(got) != 1 {
				t.Fatalf("expected 1 trace batch, got %d", len(got))
			}
			spans := got[0].ResourceSpans().At(0).ScopeSpans().At(0).Spans()
			if spans.Len() != 3 {
				t.Fatalf("expected 3 spans, got %d", spans.Len())
			}
			for i := 0; i < spans.Len(); i++ {
				span := spans.At(i)
				v, ok := span.Attributes().Get("example_key")
				if !ok {
					t.Fatalf("span %d: attribute example_key not found", i)
				}
				if v.Str() != "example_value" {
					t.Fatalf("span %d: expected example_value, got %q", i, v.Str())
				}
				if span.Name() != fmt.Sprintf("span-%d", i) {
					t.Fatalf("span %d: unexpected name %q", i, span.Name())
				}
			}
		})
	}
}

func TestHandlePassthrough(t *testing.T) {
	for _, name := range []string{"passthrough_handle", "passthrough_proto"} {
		t.Run(name, func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.Path = "testdata/" + name + "/main.wasm"
			ctx := t.Context()

			sink := new(consumertest.TracesSink)
			settings := processortest.NewNopSettings(typeStr)
			tp, err := factory.CreateTraces(ctx, settings, cfg, sink)
			if err != nil {
				t.Fatalf("failed to create traces processor: %v", err)
			}
			if err := tp.Start(ctx, componenttest.NewNopHost()); err != nil {
				t.Fatalf("failed to start processor: %v", err)
			}
			defer func() {
				if err := tp.Shutdown(ctx); err != nil {
					t.Errorf("failed to shutdown processor: %v", err)
				}
			}()

			td := generateTracesWithSpans(5)
			if err := tp.ConsumeTraces(ctx, td); err != nil {
				t.Fatalf("failed to consume traces: %v", err)
			}

			got := sink.AllTraces()
			if len(got) != 1 {
				t.Fatalf("expected 1 trace batch, got %d", len(got))
			}
			if got[0].SpanCount() != 5 {
				t.Fatalf("expected 5 spans, got %d", got[0].SpanCount())
			}
		})
	}
}
