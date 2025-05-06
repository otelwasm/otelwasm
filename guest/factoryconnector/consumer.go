package factoryconnector

import (
	"context"
	"runtime"

	"github.com/otelwasm/otelwasm/guest/imports"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var _ consumer.ConsumeLogsFunc = ConsumeLogs

func ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	imports.SetResultLogs(ld)
	runtime.KeepAlive(ld) // until ptr is no longer needed.
	return nil
}

var _ consumer.ConsumeMetricsFunc = ConsumeMetrics

func ConsumeMetrics(ctx context.Context, ld pmetric.Metrics) error {
	imports.SetResultMetrics(ld)
	runtime.KeepAlive(ld) // until ptr is no longer needed.
	return nil
}

var _ consumer.ConsumeTracesFunc = ConsumeTraces

func ConsumeTraces(ctx context.Context, ld ptrace.Traces) error {
	imports.SetResultTraces(ld)
	runtime.KeepAlive(ld) // until ptr is no longer needed.
	return nil
}
