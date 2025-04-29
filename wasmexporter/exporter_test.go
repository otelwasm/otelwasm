// Package wasmexporter provides tests for the WebAssembly exporter.
package wasmexporter

import (
	"testing"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
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

func TestCreateTracesExporter(t *testing.T) {
	// Test that the exporter can be created with the default config
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()

	// Test for traces
	settings := exportertest.NewNopSettings(typeStr)
	te, err := factory.CreateTraces(ctx, settings, cfg)
	if err != nil {
		t.Fatalf("failed to create traces exporter: %v", err)
	}
	if te == nil {
		t.Fatal("traces exporter is nil")
	}

	if err := te.Start(ctx, componenttest.NewNopHost()); err != nil {
		t.Errorf("failed to start exporter: %v", err)
	}

	if err := te.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown exporter: %v", err)
	}
}

func TestCreateMetricsExporter(t *testing.T) {
	// Test that the exporter can be created with the default config
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()

	// Test for metrics
	settings := exportertest.NewNopSettings(typeStr)
	me, err := factory.CreateMetrics(ctx, settings, cfg)
	if err != nil {
		t.Fatalf("failed to create metrics exporter: %v", err)
	}
	if me == nil {
		t.Fatal("metrics exporter is nil")
	}

	if err := me.Start(ctx, componenttest.NewNopHost()); err != nil {
		t.Errorf("failed to start exporter: %v", err)
	}

	if err := me.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown exporter: %v", err)
	}
}

func TestCreateLogsExporter(t *testing.T) {
	// Test that the exporter can be created with the default config
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()

	// Test for logs
	settings := exportertest.NewNopSettings(typeStr)
	le, err := factory.CreateLogs(ctx, settings, cfg)
	if err != nil {
		t.Fatalf("failed to create logs exporter: %v", err)
	}
	if le == nil {
		t.Fatal("logs exporter is nil")
	}

	if err := le.Start(ctx, componenttest.NewNopHost()); err != nil {
		t.Errorf("failed to start exporter: %v", err)
	}

	if err := le.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown exporter: %v", err)
	}
}

func TestExportTracesWithNopExporter(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()
	wasmExp, err := newWasmExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create wasm exporter: %v", err)
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

	// Push the traces
	err = wasmExp.pushTraces(ctx, traces)
	if err != nil {
		t.Fatalf("failed to push traces: %v", err)
	}
}

func TestExportMetricsWithNopExporter(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()
	wasmExp, err := newWasmExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create wasm exporter: %v", err)
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

	// Push the metrics
	err = wasmExp.pushMetrics(ctx, metrics)
	if err != nil {
		t.Fatalf("failed to push metrics: %v", err)
	}
}

func TestExportLogsWithNopExporter(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()
	wasmExp, err := newWasmExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create wasm exporter: %v", err)
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

	// Push the logs
	err = wasmExp.pushLogs(ctx, logs)
	if err != nil {
		t.Fatalf("failed to push logs: %v", err)
	}
}

func TestExportTracesWithStdoutExporter(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/stdout/main.wasm"
	ctx := t.Context()
	wasmExp, err := newWasmExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create wasm exporter: %v", err)
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

	// Push the traces
	err = wasmExp.pushTraces(ctx, traces)
	if err != nil {
		t.Fatalf("failed to push traces: %v", err)
	}
}

func TestExportMetricsWithStdoutExporter(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/stdout/main.wasm"
	ctx := t.Context()
	wasmExp, err := newWasmExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create wasm exporter: %v", err)
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

	// Push the metrics
	err = wasmExp.pushMetrics(ctx, metrics)
	if err != nil {
		t.Fatalf("failed to push metrics: %v", err)
	}
}

func TestExportLogsWithStdoutExporter(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/stdout/main.wasm"
	ctx := t.Context()
	wasmExp, err := newWasmExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create wasm exporter: %v", err)
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

	// Push the logs
	err = wasmExp.pushLogs(ctx, logs)
	if err != nil {
		t.Fatalf("failed to push logs: %v", err)
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

func TestShutdown(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := t.Context()
	wasmExp, err := newWasmExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create wasm exporter: %v", err)
	}

	// Shutdown the exporter
	err = wasmExp.shutdown(ctx)
	if err != nil {
		t.Fatalf("failed to shutdown exporter: %v", err)
	}
}
