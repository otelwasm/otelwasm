package plugin

type TelemetryType uint32

const (
	telemetryTypeMetrics TelemetryType = 1 << iota
	telemetryTypeLogs
	telemetryTypeTraces
)

// supportedTelemetry is a set of flags indicating the telemetry types supported by the plugin.
var supportedTelemetry TelemetryType = 0

var (
	_ func()                      = _otelwasmABIVersion010
	_ func() uint32               = _getSupportedTelemetry
	_ func() uint32               = _start
	_ func() uint32               = _shutdown
	_ func(uint32, uint32) uint32 = _consumeTraces
	_ func(uint32, uint32) uint32 = _consumeMetrics
	_ func(uint32, uint32) uint32 = _consumeLogs
	_ func()                      = _startTracesReceiver
	_ func()                      = _startMetricsReceiver
	_ func()                      = _startLogsReceiver
)

//go:wasmexport otelwasm_abi_version_0_1_0
func _otelwasmABIVersion010() {}

//go:wasmexport get_supported_telemetry
func _getSupportedTelemetry() uint32 {
	return uint32(supportedTelemetry)
}

//go:wasmexport start
func _start() uint32 {
	return 0
}

//go:wasmexport shutdown
func _shutdown() uint32 {
	return 0
}
