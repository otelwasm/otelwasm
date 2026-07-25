package wasmprocessor

import (
	"context"
	"fmt"
	"testing"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/processortest"
)

// nativeAddAttributeLogs is the native Go equivalent of the
// logs_add_attribute_{proto,handle} guests: walk every log record and upsert
// one string attribute.
func nativeAddAttributeLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		sls := rls.At(i).ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			lrs := sls.At(j).LogRecords()
			for k := 0; k < lrs.Len(); k++ {
				lrs.At(k).Attributes().PutStr("example_key", "example_value")
			}
		}
	}
	return ld, nil
}

// BenchmarkHandleVsProtoLogsNative is the native baseline for
// BenchmarkHandleVsProtoLogs: the same per-record attribute upsert running as
// a plain Go processor through the same processorhelper harness.
func BenchmarkHandleVsProtoLogsNative(b *testing.B) {
	for _, numRecords := range []int{1, 100, 1000} {
		b.Run(fmt.Sprintf("records=%d", numRecords), func(b *testing.B) {
			ctx := b.Context()
			settings := processortest.NewNopSettings(typeStr)
			lp, err := processorhelper.NewLogs(
				ctx,
				settings,
				&struct{ component.Config }{},
				consumertest.NewNop(),
				nativeAddAttributeLogs,
				processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
			)
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
		})
	}
}
