package plugin

type TelemetryType uint32

const (
	telemetryTypeMetrics TelemetryType = 1 << iota
	telemetryTypeLogs
	telemetryTypeTraces
)

// supportedTelemetry is a set of flags indicating the telemetry types supported by the plugin.
var supportedTelemetry TelemetryType = 0

var _ func() uint32 = _getSupportedTelemetry

//go:wasmexport getSupportedTelemetry
func _getSupportedTelemetry() uint32 {
	return uint32(supportedTelemetry)
}
