package wasmprocessor

import (
	"fmt"
	"testing"

	"github.com/otelwasm/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

// BenchmarkHandleVsProtoMetrics compares the handle/accessor ABI against the
// proto-serialization ABI for an identical guest workload that walks every
// gauge/sum data point and upserts one string attribute.
func BenchmarkHandleVsProtoMetrics(b *testing.B) {
	for _, mode := range []wasmplugin.RuntimeMode{wasmplugin.RuntimeModeInterpreter, wasmplugin.RuntimeModeCompiled} {
		for _, abi := range []string{"proto", "handle"} {
			for _, numDataPoints := range []int{1, 100, 1000} {
				name := fmt.Sprintf("%s/%s/datapoints=%d", mode, abi, numDataPoints)
				b.Run(name, func(b *testing.B) {
					benchmarkWasmMetrics(b, "metrics_add_attribute_"+abi, mode, numDataPoints)
				})
			}
		}
	}
}

// BenchmarkHandleVsProtoLogs compares the handle/accessor ABI against the
// proto-serialization ABI for an identical guest workload that walks every
// log record and upserts one string attribute.
func BenchmarkHandleVsProtoLogs(b *testing.B) {
	for _, mode := range []wasmplugin.RuntimeMode{wasmplugin.RuntimeModeInterpreter, wasmplugin.RuntimeModeCompiled} {
		for _, abi := range []string{"proto", "handle"} {
			for _, numRecords := range []int{1, 100, 1000} {
				name := fmt.Sprintf("%s/%s/records=%d", mode, abi, numRecords)
				b.Run(name, func(b *testing.B) {
					benchmarkWasmLogs(b, "logs_add_attribute_"+abi, mode, numRecords)
				})
			}
		}
	}
}

func benchmarkWasmMetrics(b *testing.B, guestName string, mode wasmplugin.RuntimeMode, numDataPoints int) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/" + guestName + "/main.wasm"
	cfg.RuntimeConfig.Mode = mode
	ctx := b.Context()

	settings := processortest.NewNopSettings(typeStr)
	mp, err := factory.CreateMetrics(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		b.Fatalf("failed to create metrics processor: %v", err)
	}
	if err := mp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Fatalf("failed to start processor: %v", err)
	}
	defer func() {
		if err := mp.Shutdown(ctx); err != nil {
			b.Errorf("failed to shutdown processor: %v", err)
		}
	}()

	md := generateMetricsWithDataPoints(numDataPoints)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := mp.ConsumeMetrics(ctx, md); err != nil {
			b.Fatalf("failed to consume metrics: %v", err)
		}
	}
}

func benchmarkWasmLogs(b *testing.B, guestName string, mode wasmplugin.RuntimeMode, numRecords int) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/" + guestName + "/main.wasm"
	cfg.RuntimeConfig.Mode = mode
	ctx := b.Context()

	settings := processortest.NewNopSettings(typeStr)
	lp, err := factory.CreateLogs(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		b.Fatalf("failed to create logs processor: %v", err)
	}
	if err := lp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Fatalf("failed to start processor: %v", err)
	}
	defer func() {
		if err := lp.Shutdown(ctx); err != nil {
			b.Errorf("failed to shutdown processor: %v", err)
		}
	}()

	ld := generateLogsWithRecords(numRecords)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := lp.ConsumeLogs(ctx, ld); err != nil {
			b.Fatalf("failed to consume logs: %v", err)
		}
	}
}
