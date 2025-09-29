package wasmprocessor

import (
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/otelwasm/otelwasm/wasmplugin"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

func generateExampleTraces() ptrace.Traces {
	td := ptrace.NewTraces()
	resourceSpans := td.ResourceSpans().AppendEmpty()
	resourceSpans.Resource().Attributes().PutStr("example_key", "example_value")
	span := resourceSpans.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("example_span")
	span.SetKind(ptrace.SpanKindServer)
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(1 * time.Second)))
	span.Attributes().PutStr("example_span_key", "example_span_value")
	return td
}

func BenchmarkNopProcessorWasmInterpreter(b *testing.B) {
	// Test that the processor can be created with the default config
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/nop/main.wasm"
	ctx := b.Context()

	// Test for traces
	settings := processortest.NewNopSettings(typeStr)
	tp, err := factory.CreateTraces(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		b.Fatalf("failed to create traces processor: %v", err)
	}
	if tp == nil {
		b.Fatal("traces processor is nil")
	}

	if err := tp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Errorf("failed to start processor: %v", err)
	}

	td := generateExampleTraces()

	b.Run("process traces", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := tp.ConsumeTraces(ctx, td); err != nil {
				b.Errorf("failed to consume traces: %v", err)
			}
		}
	})

	if err := tp.Shutdown(ctx); err != nil {
		b.Errorf("failed to shutdown processor: %v", err)
	}
}

func BenchmarkNopProcessorWasmCompiled(b *testing.B) {
	// Test that the processor can be created with the default config
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.RuntimeConfig = &wasmplugin.RuntimeConfig{
		// 		Type: wasmplugin.RuntimeTypeWazero,
		// 		Wazero: &wasmplugin.WazeroConfig{
		// 			Mode: wasmplugin.WazeroRuntimeModeCompiled,
		// 		},
	}
	cfg.Path = "testdata/nop/main.wasm"
	ctx := b.Context()

	// Test for traces
	settings := processortest.NewNopSettings(typeStr)
	tp, err := factory.CreateTraces(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		b.Fatalf("failed to create traces processor: %v", err)
	}
	if tp == nil {
		b.Fatal("traces processor is nil")
	}

	if err := tp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Errorf("failed to start processor: %v", err)
	}

	td := generateExampleTraces()

	b.Run("process traces", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := tp.ConsumeTraces(ctx, td); err != nil {
				b.Errorf("failed to consume traces: %v", err)
			}
		}
	})

	if err := tp.Shutdown(ctx); err != nil {
		b.Errorf("failed to shutdown processor: %v", err)
	}
}

func BenchmarkNopProcessorGo(b *testing.B) {
	factory := processortest.NewNopFactory()
	cfg := factory.CreateDefaultConfig()
	ctx := b.Context()

	// Test for traces
	settings := processortest.NewNopSettings(processortest.NopType)
	tp, err := factory.CreateTraces(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		b.Fatalf("failed to create traces processor: %v", err)
	}
	if tp == nil {
		b.Fatal("traces processor is nil")
	}

	if err := tp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Errorf("failed to start processor: %v", err)
	}

	td := generateExampleTraces()

	b.Run("process traces", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := tp.ConsumeTraces(ctx, td); err != nil {
				b.Errorf("failed to consume traces: %v", err)
			}
		}
	})

	if err := tp.Shutdown(ctx); err != nil {
		b.Errorf("failed to shutdown processor: %v", err)
	}
}

func BenchmarkAttributesProcessorWasmInterpreter(b *testing.B) {
	// Test that the processor can be created with the default config
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Path = "testdata/attributesprocessor/main.wasm"
	cfg.PluginConfig = wasmplugin.PluginConfig{
		"actions": []map[string]string{
			{
				"key":    "key",
				"value":  "value",
				"action": "insert",
			},
		},
	}
	ctx := b.Context()

	// Test for traces
	settings := processortest.NewNopSettings(typeStr)
	tp, err := factory.CreateTraces(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		b.Fatalf("failed to create traces processor: %v", err)
	}
	if tp == nil {
		b.Fatal("traces processor is nil")
	}

	if err := tp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Errorf("failed to start processor: %v", err)
	}

	td := generateExampleTraces()

	b.Run("process traces", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := tp.ConsumeTraces(ctx, td); err != nil {
				b.Errorf("failed to consume traces: %v", err)
			}
		}
	})

	if err := tp.Shutdown(ctx); err != nil {
		b.Errorf("failed to shutdown processor: %v", err)
	}
}

// TODO(tsuzu): Investigate why this crashes on arm64 (related to #64)
// func BenchmarkAttributesProcessorWasmCompiled(b *testing.B) {
// 	// Test that the processor can be created with the default config
// 	factory := NewFactory()
// 	cfg := factory.CreateDefaultConfig().(*Config)
// 	cfg.Path = "testdata/attributesprocessor/main.wasm"
// 	cfg.PluginConfig = wasmplugin.PluginConfig{
// 		"actions": []map[string]string{
// 			{
// 				"key":    "key",
// 				"value":  "value",
// 				"action": "insert",
// 	// 		},
// // 		},
// 	}
// 	cfg.RuntimeConfig = &wasmplugin.RuntimeConfig{
// 		Type: wasmplugin.RuntimeTypeWazero,
// 		Wazero: &wasmplugin.WazeroConfig{
// 			Mode: wasmplugin.WazeroRuntimeModeCompiled,
// 		},
// 	}
// 	ctx := b.Context()

// 	// Test for traces
// 	settings := processortest.NewNopSettings(typeStr)
// 	tp, err := factory.CreateTraces(ctx, settings, cfg, consumertest.NewNop())
// 	if err != nil {
// 		b.Fatalf("failed to create traces processor: %v", err)
// 	}
// 	if tp == nil {
// 		b.Fatal("traces processor is nil")
// 	}

// 	if err := tp.Start(ctx, componenttest.NewNopHost()); err != nil {
// 		b.Errorf("failed to start processor: %v", err)
// 	}

// 	td := generateExampleTraces()

// 	b.Run("process traces", func(b *testing.B) {
// 		for i := 0; i < b.N; i++ {
// 			if err := tp.ConsumeTraces(ctx, td); err != nil {
// 				b.Errorf("failed to consume traces: %v", err)
// 			}
// 		}
// 	})

// 	if err := tp.Shutdown(ctx); err != nil {
// 		b.Errorf("failed to shutdown processor: %v", err)
// 	}
// }

func BenchmarkAttributesProcessorGo(b *testing.B) {
	// Test that the processor can be created with the default config
	factory := attributesprocessor.NewFactory()
	cfg := factory.CreateDefaultConfig().(*attributesprocessor.Config)

	err := mapstructure.Decode(map[string]any{
		"actions": []map[string]string{
			{
				"key":    "key",
				"value":  "value",
				"action": "insert",
			},
		},
	}, &cfg)
	if err != nil {
		b.Fatalf("failed to decode config: %v", err)
	}

	ctx := b.Context()

	// Test for traces
	settings := processortest.NewNopSettings(factory.Type())
	tp, err := factory.CreateTraces(ctx, settings, cfg, consumertest.NewNop())
	if err != nil {
		b.Fatalf("failed to create traces processor: %v", err)
	}
	if tp == nil {
		b.Fatal("traces processor is nil")
	}

	if err := tp.Start(ctx, componenttest.NewNopHost()); err != nil {
		b.Errorf("failed to start processor: %v", err)
	}

	td := generateExampleTraces()

	b.Run("process traces", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := tp.ConsumeTraces(ctx, td); err != nil {
				b.Errorf("failed to consume traces: %v", err)
			}
		}
	})

	if err := tp.Shutdown(ctx); err != nil {
		b.Errorf("failed to shutdown processor: %v", err)
	}
}
