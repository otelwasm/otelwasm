package wasmprocessor

import (
	"testing"

	"github.com/otelwasm/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
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
	cfg.Path = "testdata/attributesprocessor/main.wasm"
	cfg.PluginConfig = wasmplugin.PluginConfig{
		"actions": []map[string]string{
			{
				"key":    "phase_a",
				"value":  "true",
				"action": "insert",
			},
		},
	}
	ctx := t.Context()

	// Test for traces
	settings := processortest.NewNopSettings(typeStr)
	tp, err := factory.CreateTraces(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		t.Fatalf("failed to create traces processor: %v", err)
	}
	if tp == nil {
		t.Fatal("traces processor is nil")
	}

	if err := tp.Start(ctx, componenttest.NewNopHost()); err != nil {
		t.Errorf("failed to start processor: %v", err)
	}

	if err := tp.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown processor: %v", err)
	}
}

func TestCreateMetricsProcessor(t *testing.T) {
	// Test that the processor can be created with the default config
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()

	// Test for metrics
	settings := processortest.NewNopSettings(typeStr)
	mp, err := factory.CreateMetrics(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		t.Fatalf("failed to create metrics processor: %v", err)
	}
	if mp == nil {
		t.Fatal("metrics processor is nil")
	}

	if err := mp.Start(ctx, componenttest.NewNopHost()); err != nil {
		t.Errorf("failed to start processor: %v", err)
	}

	if err := mp.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown processor: %v", err)
	}
}

func TestCreateLogsProcessor(t *testing.T) {
	// Test that the processor can be created with the default config
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()

	// Test for logs
	settings := processortest.NewNopSettings(typeStr)
	lp, err := factory.CreateLogs(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		t.Fatalf("failed to create logs processor: %v", err)
	}
	if lp == nil {
		t.Fatal("logs processor is nil")
	}

	if err := lp.Start(ctx, componenttest.NewNopHost()); err != nil {
		t.Errorf("failed to start processor: %v", err)
	}

	if err := lp.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown processor: %v", err)
	}
}

func TestProcessTracesWithNopProcessor(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/attributesprocessor/main.wasm"
	cfg.PluginConfig = wasmplugin.PluginConfig{
		"actions": []map[string]string{
			{
				"key":    "phase_a",
				"value":  "true",
				"action": "insert",
			},
		},
	}
	ctx := t.Context()
	wasmProc, err := newWasmTracesProcessor(ctx, cfg)
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
	processedTraces, err := wasmProc.processTraces(ctx, traces)
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

func TestProcessTracesWithCurlProcessor(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/attributesprocessor/main.wasm"
	cfg.PluginConfig = wasmplugin.PluginConfig{
		"actions": []map[string]string{
			{
				"key":    "phase_a",
				"value":  "true",
				"action": "insert",
			},
		},
	}
	ctx := t.Context()
	wasmProc, err := newWasmTracesProcessor(ctx, cfg)
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
	_, err = wasmProc.processTraces(ctx, traces)
	if err != nil {
		t.Fatalf("failed to process traces: %v", err)
	}
}

func TestProcessMetricsWithNopProcessor(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()
	wasmProc, err := newWasmMetricsProcessor(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create wasm processor: %v", err)
	}

	// Create test metrics with 1 resource, 1 scope, and 1 metric
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "test-service")
	ilm := rm.ScopeMetrics().AppendEmpty()
	ilm.Scope().SetName("test-scope")
	metric := ilm.Metrics().AppendEmpty()
	metric.SetName("test-metric")
	metric.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(42)

	// Process the metrics
	processedMetrics, err := wasmProc.processMetrics(ctx, metrics)
	if err != nil {
		t.Fatalf("failed to process metrics: %v", err)
	}

	// Verify the processed metrics
	processedRM := processedMetrics.ResourceMetrics()
	if processedRM.Len() != 1 {
		t.Fatalf("expected 1 resource metric, got %d", processedRM.Len())
	}

	processedResource := processedRM.At(0).Resource()
	if val, ok := processedResource.Attributes().Get("service.name"); !ok || val.Str() != "test-service" {
		t.Errorf("expected service.name to be 'test-service', got %v", val)
	}

	processedILM := processedRM.At(0).ScopeMetrics()
	if processedILM.Len() != 1 {
		t.Fatalf("expected 1 scope metric, got %d", processedILM.Len())
	}

	processedMetric := processedILM.At(0).Metrics().At(0)
	if processedMetric.Name() != "test-metric" {
		t.Errorf("expected metric name to be 'test-metric', got %s", processedMetric.Name())
	}
	if processedMetric.Gauge().DataPoints().At(0).IntValue() != 42 {
		t.Errorf("expected metric value to be 42, got %d", processedMetric.Gauge().DataPoints().At(0).IntValue())
	}
}

func TestProcessLogsWithNopProcessor(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()
	wasmProc, err := newWasmLogsProcessor(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create wasm processor: %v", err)
	}

	// Create test logs with 1 resource, 1 scope, and 1 log record
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("service.name", "test-service")
	sl := rl.ScopeLogs().AppendEmpty()
	sl.Scope().SetName("test-scope")
	logRecord := sl.LogRecords().AppendEmpty()
	logRecord.SetSeverityText("INFO")
	logRecord.Body().SetStr("test message")

	// Process the logs
	processedLogs, err := wasmProc.processLogs(ctx, logs)
	if err != nil {
		t.Fatalf("failed to process logs: %v", err)
	}

	// Verify the processed logs
	processedRL := processedLogs.ResourceLogs()
	if processedRL.Len() != 1 {
		t.Fatalf("expected 1 resource log, got %d", processedRL.Len())
	}

	processedResource := processedRL.At(0).Resource()
	if val, ok := processedResource.Attributes().Get("service.name"); !ok || val.Str() != "test-service" {
		t.Errorf("expected service.name to be 'test-service', got %v", val)
	}

	processedSL := processedRL.At(0).ScopeLogs()
	if processedSL.Len() != 1 {
		t.Fatalf("expected 1 scope log, got %d", processedSL.Len())
	}

	processedLogRecord := processedSL.At(0).LogRecords().At(0)
	if processedLogRecord.SeverityText() != "INFO" {
		t.Errorf("expected severity text to be 'INFO', got %s", processedLogRecord.SeverityText())
	}
	if processedLogRecord.Body().Str() != "test message" {
		t.Errorf("expected log message to be 'test message', got %s", processedLogRecord.Body().Str())
	}
}

func TestProcessTracesWithAddNewAttributeProcessor(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/attributesprocessor/main.wasm"
	cfg.PluginConfig = wasmplugin.PluginConfig{
		"actions": []map[string]string{
			{
				"key":    "new-attribute",
				"value":  "new-value",
				"action": "insert",
			},
		},
	}
	ctx := t.Context()
	wasmProc, err := newWasmTracesProcessor(ctx, cfg)
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
	processedTraces, err := wasmProc.processTraces(ctx, traces)
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
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	if err := cfg.Validate(); err != nil {
		t.Errorf("config validation failed: %v", err)
	}
}
