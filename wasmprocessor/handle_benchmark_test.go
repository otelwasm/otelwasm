package wasmprocessor

import (
	"fmt"
	"testing"

	"github.com/otelwasm/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

// BenchmarkHandleVsProto compares the handle/accessor ABI against the
// proto-serialization ABI for identical guest workloads:
//   - add_attribute: walks every span and upserts one string attribute
//   - passthrough: walks every span reading name/kind, data unchanged
func BenchmarkHandleVsProto(b *testing.B) {
	modes := []wasmplugin.RuntimeMode{
		wasmplugin.RuntimeModeInterpreter,
		wasmplugin.RuntimeModeCompiled,
	}
	workloads := []string{"add_attribute", "passthrough"}
	abis := []string{"proto", "handle"}
	spanCounts := []int{1, 100, 1000}

	for _, mode := range modes {
		for _, workload := range workloads {
			for _, abi := range abis {
				for _, numSpans := range spanCounts {
					name := fmt.Sprintf("%s/%s/%s/spans=%d", mode, workload, abi, numSpans)
					b.Run(name, func(b *testing.B) {
						benchmarkWasmTraces(b, workload+"_"+abi, mode, numSpans)
					})
				}
			}
		}
	}
}

func benchmarkWasmTraces(b *testing.B, guestName string, mode wasmplugin.RuntimeMode, numSpans int) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/" + guestName + "/main.wasm"
	cfg.RuntimeConfig.Mode = mode
	ctx := b.Context()

	settings := processortest.NewNopSettings(typeStr)
	tp, err := factory.CreateTraces(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		b.Fatalf("failed to create traces processor: %v", err)
	}
	if err := tp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Fatalf("failed to start processor: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			b.Errorf("failed to shutdown processor: %v", err)
		}
	}()

	td := generateTracesWithSpans(numSpans)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := tp.ConsumeTraces(ctx, td); err != nil {
			b.Fatalf("failed to consume traces: %v", err)
		}
	}
}
