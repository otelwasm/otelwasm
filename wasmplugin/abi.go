package wasmplugin

import "github.com/tetratelabs/wazero/api"

const abiVersionV1MarkerExport = "otelwasm_abi_version_0_1_0"

// ABIVersion represents the detected plugin ABI.
type ABIVersion uint8

const (
	// ABIUnknown indicates that no known ABI marker was exported.
	ABIUnknown ABIVersion = iota
	// ABIV1 indicates the plugin exports the ABI v1 marker.
	ABIV1
)

func (v ABIVersion) String() string {
	switch v {
	case ABIV1:
		return "v1"
	case ABIUnknown:
		return "unknown"
	default:
		return "invalid"
	}
}

func detectABIVersion(mod api.Module) ABIVersion {
	if mod == nil {
		return ABIUnknown
	}
	if mod.ExportedFunction(abiVersionV1MarkerExport) != nil {
		return ABIV1
	}
	return ABIUnknown
}
