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
	_ func()        = _abiVersionV1
	_ func() uint32 = _getSupportedTelemetry
	_ func() uint32 = _start
	_ func() uint32 = _shutdown
)

//go:wasmexport abi_version_v1
func _abiVersionV1() {}

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
