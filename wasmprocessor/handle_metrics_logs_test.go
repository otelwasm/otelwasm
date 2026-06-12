package wasmprocessor

import (
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processortest"
)

// generateMetricsWithDataPoints returns metrics with a single resource/scope
// and the given number of gauge metrics, one data point each with a few
// attributes.
func generateMetricsWithDataPoints(numDataPoints int) pmetric.Metrics {
	md := pmetric.NewMetrics()
	resourceMetrics := md.ResourceMetrics().AppendEmpty()
	resourceMetrics.Resource().Attributes().PutStr("service.name", "benchmark-service")
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	scopeMetrics.Scope().SetName("benchmark-scope")
	ts := pcommon.NewTimestampFromTime(time.Unix(1700000000, 0))
	for i := 0; i < numDataPoints; i++ {
		m := scopeMetrics.Metrics().AppendEmpty()
		m.SetName(fmt.Sprintf("metric-%d", i))
		m.SetUnit("1")
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetTimestamp(ts)
		dp.SetDoubleValue(float64(i) * 1.5)
		attrs := dp.Attributes()
		attrs.PutStr("http.method", "GET")
		attrs.PutStr("http.route", "/api/v1/items/:id")
		attrs.PutStr("net.peer.name", "upstream.example.com")
		attrs.PutInt("http.status_code", 200)
		attrs.PutBool("error", false)
	}
	return md
}

// generateLogsWithRecords returns logs with a single resource/scope and the
// given number of log records with a few attributes.
func generateLogsWithRecords(numRecords int) plog.Logs {
	ld := plog.NewLogs()
	resourceLogs := ld.ResourceLogs().AppendEmpty()
	resourceLogs.Resource().Attributes().PutStr("service.name", "benchmark-service")
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName("benchmark-scope")
	ts := pcommon.NewTimestampFromTime(time.Unix(1700000000, 0))
	for i := 0; i < numRecords; i++ {
		lr := scopeLogs.LogRecords().AppendEmpty()
		lr.SetTimestamp(ts)
		lr.SetSeverityNumber(plog.SeverityNumberInfo)
		lr.SetSeverityText("INFO")
		lr.Body().SetStr(fmt.Sprintf("log message %d with some payload text", i))
		attrs := lr.Attributes()
		attrs.PutStr("http.method", "GET")
		attrs.PutStr("http.route", "/api/v1/items/:id")
		attrs.PutStr("net.peer.name", "upstream.example.com")
		attrs.PutInt("http.status_code", 200)
		attrs.PutBool("error", false)
	}
	return ld
}

func TestHandleMetricsAddAttribute(t *testing.T) {
	for _, name := range []string{"metrics_add_attribute_handle", "metrics_add_attribute_proto"} {
		t.Run(name, func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.Path = "testdata/" + name + "/main.wasm"
			ctx := t.Context()

			sink := new(consumertest.MetricsSink)
			settings := processortest.NewNopSettings(typeStr)
			mp, err := factory.CreateMetrics(ctx, settings, cfg, sink)
			if err != nil {
				t.Fatalf("failed to create metrics processor: %v", err)
			}
			if err := mp.Start(ctx, componenttest.NewNopHost()); err != nil {
				t.Fatalf("failed to start processor: %v", err)
			}
			defer func() {
				if err := mp.Shutdown(ctx); err != nil {
					t.Errorf("failed to shutdown processor: %v", err)
				}
			}()

			md := generateMetricsWithDataPoints(3)
			if err := mp.ConsumeMetrics(ctx, md); err != nil {
				t.Fatalf("failed to consume metrics: %v", err)
			}

			got := sink.AllMetrics()
			if len(got) != 1 {
				t.Fatalf("expected 1 metrics batch, got %d", len(got))
			}
			metrics := got[0].ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
			if metrics.Len() != 3 {
				t.Fatalf("expected 3 metrics, got %d", metrics.Len())
			}
			for i := 0; i < metrics.Len(); i++ {
				m := metrics.At(i)
				if m.Name() != fmt.Sprintf("metric-%d", i) {
					t.Fatalf("metric %d: unexpected name %q", i, m.Name())
				}
				dp := m.Gauge().DataPoints().At(0)
				v, ok := dp.Attributes().Get("example_key")
				if !ok {
					t.Fatalf("metric %d: attribute example_key not found", i)
				}
				if v.Str() != "example_value" {
					t.Fatalf("metric %d: expected example_value, got %q", i, v.Str())
				}
				if dp.DoubleValue() != float64(i)*1.5 {
					t.Fatalf("metric %d: unexpected value %v", i, dp.DoubleValue())
				}
			}
		})
	}
}

func TestHandleLogsAddAttribute(t *testing.T) {
	for _, name := range []string{"logs_add_attribute_handle", "logs_add_attribute_proto"} {
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
				if lr.Body().Str() != fmt.Sprintf("log message %d with some payload text", i) {
					t.Fatalf("record %d: unexpected body %q", i, lr.Body().Str())
				}
			}
		})
	}
}
