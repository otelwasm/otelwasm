package wasmprocessor

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/otelwasm/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"
)

// generateBenchmarkLogs builds a moderately realistic logs payload:
// 10 resources (services) x 10 log records with typical resource,
// HTTP and user attributes. 100 log records in total per batch.
func generateBenchmarkLogs() plog.Logs {
	ld := plog.NewLogs()
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	severities := []struct {
		num  plog.SeverityNumber
		text string
	}{
		{plog.SeverityNumberInfo, "INFO"},
		{plog.SeverityNumberWarn, "WARN"},
		{plog.SeverityNumberError, "ERROR"},
	}

	for i := 0; i < 10; i++ {
		rl := ld.ResourceLogs().AppendEmpty()
		res := rl.Resource().Attributes()
		res.PutStr("service.name", fmt.Sprintf("checkout-service-%d", i))
		res.PutStr("service.version", "1.42.3")
		res.PutStr("service.namespace", "ecommerce")
		res.PutStr("host.name", fmt.Sprintf("ip-10-0-12-%d.ec2.internal", 100+i))
		res.PutStr("k8s.namespace.name", "production")
		res.PutStr("k8s.pod.name", fmt.Sprintf("checkout-service-7d9f8b6c5-%05d", i))
		res.PutStr("cloud.region", "ap-northeast-1")

		sl := rl.ScopeLogs().AppendEmpty()
		sl.Scope().SetName("github.com/example/checkout/middleware")
		sl.Scope().SetVersion("0.9.1")

		for j := 0; j < 10; j++ {
			lr := sl.LogRecords().AppendEmpty()
			sev := severities[j%len(severities)]
			ts := now.Add(time.Duration(i*10+j) * time.Millisecond)
			lr.SetTimestamp(pcommon.NewTimestampFromTime(ts))
			lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(ts.Add(2 * time.Millisecond)))
			lr.SetSeverityNumber(sev.num)
			lr.SetSeverityText(sev.text)
			lr.Body().SetStr(fmt.Sprintf(
				"request completed: method=POST path=/api/v1/checkout status=%d latency=%dms upstream=payment-gateway retry=%t",
				200+(j%3)*100, 12+j*7, j%4 == 0,
			))

			attrs := lr.Attributes()
			attrs.PutStr("http.method", "POST")
			attrs.PutStr("http.url", fmt.Sprintf("https://shop.example.com/api/v1/checkout?session=%08d&cart=%d", i*1000+j, j))
			attrs.PutInt("http.status_code", int64(200+(j%3)*100))
			attrs.PutStr("http.user_agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
			attrs.PutStr("user.email", fmt.Sprintf("user%04d@example.com", i*100+j))
			attrs.PutStr("request.id", fmt.Sprintf("6f9a2b18-4c3d-4e5f-8a7b-%012d", i*1000+j))
			attrs.PutDouble("duration_ms", float64(12+j*7)+0.25)
			attrs.PutBool("retry", j%4 == 0)
		}
	}

	return ld
}

// benchmarkLogsActions is a typical enrichment + masking pipeline:
// two static upserts, one PII hash and one regex extraction.
// All actions perform the same amount of work on every iteration,
// so in-place mutation by the native processor doesn't skew results.
func benchmarkLogsActions() []map[string]any {
	return []map[string]any{
		{"key": "processed.by", "value": "otelwasm-bench", "action": "upsert"},
		{"key": "telemetry.pipeline", "value": "logs", "action": "upsert"},
		{"key": "user.email", "action": "hash"},
		{"key": "http.url", "pattern": `^(?P<url_scheme>https?)://(?P<url_host>[^/]+)(?P<url_path>/[^?]*)`, "action": "extract"},
	}
}

// benchmarkWasmLogsAttributes runs the attributesprocessor wasm guest on the
// given runtime configuration against the benchmark logs payload.
func benchmarkWasmLogsAttributes(b *testing.B, path string, name wasmplugin.RuntimeName, mode wasmplugin.RuntimeMode) {
	b.Helper()

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = path
	cfg.RuntimeConfig.Name = name
	cfg.RuntimeConfig.Mode = mode
	cfg.PluginConfig = wasmplugin.PluginConfig{
		"actions": benchmarkLogsActions(),
	}
	ctx := b.Context()

	settings := processortest.NewNopSettings(typeStr)
	lp, err := factory.CreateLogs(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		b.Fatalf("failed to create logs processor: %v", err)
	}

	if err := lp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Fatalf("failed to start processor: %v", err)
	}

	ld := generateBenchmarkLogs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := lp.ConsumeLogs(ctx, ld); err != nil {
			b.Fatalf("failed to consume logs: %v", err)
		}
	}
	b.StopTimer()

	if err := lp.Shutdown(ctx); err != nil {
		b.Errorf("failed to shutdown processor: %v", err)
	}
}

func BenchmarkLogsAttributesProcessorGo(b *testing.B) {
	factory := attributesprocessor.NewFactory()
	cfg := factory.CreateDefaultConfig().(*attributesprocessor.Config)

	err := mapstructure.Decode(map[string]any{
		"actions": benchmarkLogsActions(),
	}, &cfg)
	if err != nil {
		b.Fatalf("failed to decode config: %v", err)
	}

	ctx := b.Context()

	settings := processortest.NewNopSettings(factory.Type())
	lp, err := factory.CreateLogs(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		b.Fatalf("failed to create logs processor: %v", err)
	}

	if err := lp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Fatalf("failed to start processor: %v", err)
	}

	ld := generateBenchmarkLogs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := lp.ConsumeLogs(ctx, ld); err != nil {
			b.Fatalf("failed to consume logs: %v", err)
		}
	}
	b.StopTimer()

	if err := lp.Shutdown(ctx); err != nil {
		b.Errorf("failed to shutdown processor: %v", err)
	}
}

func BenchmarkLogsAttributesProcessorWazeroInterpreter(b *testing.B) {
	benchmarkWasmLogsAttributes(b, "testdata/attributesprocessor/main.wasm",
		wasmplugin.RuntimeNameWazero, wasmplugin.RuntimeModeInterpreter)
}

// TODO(tsuzu): The wazero compiler backend crashes with the attributesprocessor
// guest on arm64 (see #64 and the commented-out
// BenchmarkAttributesProcessorWasmCompiled in benchmark_test.go).
// Enable this once the wazero issue is resolved.
// func BenchmarkLogsAttributesProcessorWazeroCompiled(b *testing.B) {
// 	benchmarkWasmLogsAttributes(b, "testdata/attributesprocessor/main.wasm",
// 		wasmplugin.RuntimeNameWazero, wasmplugin.RuntimeModeCompiled)
// }
