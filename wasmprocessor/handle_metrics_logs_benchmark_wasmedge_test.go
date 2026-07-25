//go:build wasmedge

package wasmprocessor

import (
	"fmt"
	"os"
	"testing"

	"github.com/otelwasm/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

// BenchmarkHandleVsProtoLogsWasmEdge mirrors BenchmarkHandleVsProtoLogs on
// the WasmEdge runtime. The "aot" executions use guests precompiled with the
// WasmEdge AOT compiler (main-aot.wasm, see the benchmark-wasmedge Makefile
// target) and are skipped when the precompiled binary is missing.
func BenchmarkHandleVsProtoLogsWasmEdge(b *testing.B) {
	for _, exec := range []string{"interpreter", "aot"} {
		for _, abi := range []string{"proto", "handle"} {
			for _, numRecords := range []int{1, 100, 1000} {
				name := fmt.Sprintf("%s/%s/records=%d", exec, abi, numRecords)
				b.Run(name, func(b *testing.B) {
					file := "main.wasm"
					if exec == "aot" {
						file = "main-aot.wasm"
					}
					path := "testdata/logs_add_attribute_" + abi + "/" + file
					if _, err := os.Stat(path); err != nil {
						b.Skipf("guest not found at %s; generate it with: wasmedge compile --optimize 2", path)
					}
					benchmarkWasmEdgeLogs(b, path, numRecords)
				})
			}
		}
	}
}

func benchmarkWasmEdgeLogs(b *testing.B, path string, numRecords int) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = path
	cfg.RuntimeConfig.Name = wasmplugin.RuntimeNameWasmEdge
	cfg.RuntimeConfig.Mode = ""
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
