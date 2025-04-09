package wasmprocessor

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestCreateDefaultConfig(t *testing.T) {
	// Test that the default config can be created and cast to the correct type
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	if cfg == nil {
		t.Fatal("failed to create default config")
	}

	if err := componenttest.CheckConfigStruct(cfg); err != nil {
		t.Errorf("config failed structure validation: %v", err)
	}

	_, ok := cfg.(*Config)
	if !ok {
		t.Error("config is not the correct type")
	}
}

func TestCreateTracesProcessor(t *testing.T) {
	// Test that the processor can be created with the default config
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"

	// Test for traces
	settings := processortest.NewNopSettings(typeStr)
	tp, err := factory.CreateTraces(context.Background(), settings, cfg, consumertest.NewNop())
	if err != nil {
		t.Fatalf("failed to create traces processor: %v", err)
	}
	if tp == nil {
		t.Fatal("traces processor is nil")
	}

	if err := tp.Start(context.Background(), componenttest.NewNopHost()); err != nil {
		t.Errorf("failed to start processor: %v", err)
	}

	if err := tp.Shutdown(context.Background()); err != nil {
		t.Errorf("failed to shutdown processor: %v", err)
	}
}

func TestProcessTracesWithNopProcessor(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	wasmProc, err := newWasmProcessor(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create wasm processor: %v", err)
	}

	// Create test traces with 1 resource, 1 scope, and 1 span
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "test-service")
	ss := rs.ScopeSpans().AppendEmpty()
	ss.Scope().SetName("test-scope")
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	span.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})

	// Process the traces
	processedTraces, err := wasmProc.processTraces(context.Background(), traces)
	if err != nil {
		t.Fatalf("failed to process traces: %v", err)
	}

	// Verify the processed traces
	processedRS := processedTraces.ResourceSpans()
	if processedRS.Len() != 1 {
		t.Fatalf("expected 1 resource span, got %d", processedRS.Len())
	}

	processedResource := processedRS.At(0).Resource()
	if val, ok := processedResource.Attributes().Get("service.name"); !ok || val.Str() != "test-service" {
		t.Errorf("expected service.name to be 'test-service', got %v", val)
	}

	processedSS := processedRS.At(0).ScopeSpans()
	if processedSS.Len() != 1 {
		t.Fatalf("expected 1 scope span, got %d", processedSS.Len())
	}

	processedSpan := processedSS.At(0).Spans().At(0)
	if processedSpan.Name() != "test-span" {
		t.Errorf("expected span name to be 'test-span', got %s", processedSpan.Name())
	}
}

func TestProcessTracesWithAddNewAttributeProcessor(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/add_new_attribute/main.wasm"
	wasmProc, err := newWasmProcessor(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create wasm processor: %v", err)
	}

	// Create test traces with 1 resource, 1 scope, and 1 span
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "test-service")
	ss := rs.ScopeSpans().AppendEmpty()
	ss.Scope().SetName("test-scope")
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	span.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})

	// Verify the attribute in the span doesn't exist before processing
	if val, ok := span.Attributes().Get("new-attribute"); ok {
		t.Errorf("expected new-attribute to not exist, got %v", val)
	}

	// Process the traces
	processedTraces, err := wasmProc.processTraces(context.Background(), traces)
	if err != nil {
		t.Fatalf("failed to process traces: %v", err)
	}

	// Verify the processed traces
	processedRS := processedTraces.ResourceSpans()
	if processedRS.Len() != 1 {
		t.Fatalf("expected 1 resource span, got %d", processedRS.Len())
	}

	processedResource := processedRS.At(0).Resource()
	if val, ok := processedResource.Attributes().Get("service.name"); !ok || val.Str() != "test-service" {
		t.Errorf("expected service.name to be 'test-service', got %v", val)
	}

	// Verify attributes
	processedSS := processedRS.At(0).ScopeSpans()
	if processedSS.Len() != 1 {
		t.Fatalf("expected 1 scope span, got %d", processedSS.Len())
	}
	processedSpan := processedSS.At(0).Spans().At(0)
	if processedSpan.Name() != "test-span" {
		t.Errorf("expected span name to be 'test-span', got %s", processedSpan.Name())
	}
	if val, ok := processedSpan.Attributes().Get("new-attribute"); !ok || val.Str() != "new-value" {
		t.Errorf("expected new-attribute to be 'new-value', got %v", val)
	}
}

func TestConfigValidate(t *testing.T) {
	// Test that the config validation works as expected
	cfg := &Config{}
	cfg.Path = "testdata/nop/main.wasm"
	if err := cfg.Validate(); err != nil {
		t.Errorf("config validation failed: %v", err)
	}
}
