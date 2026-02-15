package wasmplugin

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

func FuzzConsumeTracesBoundary(f *testing.F) {
	f.Add([]byte("seed-span"), uint8(1), false)
	f.Add([]byte(""), uint8(0), true)
	f.Add([]byte("another-seed"), uint8(4), false)

	f.Fuzz(func(t *testing.T, rawName []byte, spanCountSeed uint8, empty bool) {
		if len(rawName) > 128 {
			rawName = rawName[:128]
		}

		p := newPushTestPlugin(t, buildTestModule(true, []wasmFunctionSpec{
			{name: allocFunction, typeIndex: wasmTypeFuncI32ToI32, returnValue: uint32Ptr(16)},
			{name: consumeTracesFunction, typeIndex: wasmTypeFuncI32I32ToI32, returnValue: uint32Ptr(0)},
		}), []string{consumeTracesFunction})

		td := newFuzzTraces(rawName, spanCountSeed, empty)
		expectedSpanCount := td.SpanCount()
		out, err := p.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Fatalf("ConsumeTraces returned unexpected error: %v", err)
		}

		if out.SpanCount() != expectedSpanCount {
			t.Fatalf("expected span count %d, got %d", expectedSpanCount, out.SpanCount())
		}
	})
}

func newFuzzTraces(rawName []byte, spanCountSeed uint8, empty bool) ptrace.Traces {
	td := ptrace.NewTraces()
	if empty {
		return td
	}

	spanCount := int(spanCountSeed%8) + 1
	name := string(rawName)
	if name == "" {
		name = "span"
	}

	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty().Spans()
	for i := 0; i < spanCount; i++ {
		span := ss.AppendEmpty()
		span.SetName(name)
	}
	return td
}
