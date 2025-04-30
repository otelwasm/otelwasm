package plugin

import (
	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/logsexporter"
	"github.com/musaprg/otelwasm/guest/logsprocessor"
	"github.com/musaprg/otelwasm/guest/logsreceiver"
	"github.com/musaprg/otelwasm/guest/metricsexporter"
	"github.com/musaprg/otelwasm/guest/metricsprocessor"
	"github.com/musaprg/otelwasm/guest/metricsreceiver"
	"github.com/musaprg/otelwasm/guest/tracesexporter"
	"github.com/musaprg/otelwasm/guest/tracesprocessor"
	"github.com/musaprg/otelwasm/guest/tracesreceiver"
)

func Set(plugin api.Plugin) {
	if plugin, ok := plugin.(api.TracesProcessor); ok {
		tracesprocessor.SetPlugin(plugin)
		supportedTelemetry |= telemetryTypeTraces
	}
	if plugin, ok := plugin.(api.MetricsProcessor); ok {
		metricsprocessor.SetPlugin(plugin)
		supportedTelemetry |= telemetryTypeMetrics
	}
	if plugin, ok := plugin.(api.LogsProcessor); ok {
		logsprocessor.SetPlugin(plugin)
		supportedTelemetry |= telemetryTypeLogs
	}
	if plugin, ok := plugin.(api.TracesExporter); ok {
		tracesexporter.SetPlugin(plugin)
		supportedTelemetry |= telemetryTypeTraces
	}
	if plugin, ok := plugin.(api.MetricsExporter); ok {
		metricsexporter.SetPlugin(plugin)
		supportedTelemetry |= telemetryTypeMetrics
	}
	if plugin, ok := plugin.(api.LogsExporter); ok {
		logsexporter.SetPlugin(plugin)
		supportedTelemetry |= telemetryTypeLogs
	}
	if plugin, ok := plugin.(api.MetricsReceiver); ok {
		metricsreceiver.SetPlugin(plugin)
		supportedTelemetry |= telemetryTypeMetrics
	}
	if plugin, ok := plugin.(api.LogsReceiver); ok {
		logsreceiver.SetPlugin(plugin)
		supportedTelemetry |= telemetryTypeLogs
	}
	if plugin, ok := plugin.(api.TracesReceiver); ok {
		tracesreceiver.SetPlugin(plugin)
		supportedTelemetry |= telemetryTypeTraces
	}

	// TODO: panic of return error
}
