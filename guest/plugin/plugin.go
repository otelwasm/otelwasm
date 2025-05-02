package plugin

import (
	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/logsexporter"
	"github.com/otelwasm/otelwasm/guest/logsprocessor"
	"github.com/otelwasm/otelwasm/guest/logsreceiver"
	"github.com/otelwasm/otelwasm/guest/metricsexporter"
	"github.com/otelwasm/otelwasm/guest/metricsprocessor"
	"github.com/otelwasm/otelwasm/guest/metricsreceiver"
	"github.com/otelwasm/otelwasm/guest/tracesexporter"
	"github.com/otelwasm/otelwasm/guest/tracesprocessor"
	"github.com/otelwasm/otelwasm/guest/tracesreceiver"
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
