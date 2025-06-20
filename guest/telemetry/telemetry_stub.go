//go:build !wasm

package telemetry

// TelemetrySettings represents telemetry settings received from the host
type TelemetrySettings struct {
	// Resource attributes as a map
	ResourceAttributes map[string]interface{} `json:"resource_attributes"`
	// Service name extracted from resource
	ServiceName string `json:"service_name"`
	// Service version extracted from resource  
	ServiceVersion string `json:"service_version"`
	// Component ID information
	ComponentID map[string]string `json:"component_id"`
}

// GetTelemetrySettings retrieves telemetry settings from the host (no-op for non-WASM)
func GetTelemetrySettings() (*TelemetrySettings, error) {
	return &TelemetrySettings{
		ResourceAttributes: make(map[string]interface{}),
		ServiceName:        "",
		ServiceVersion:     "",
		ComponentID:        make(map[string]string),
	}, nil
}

// GetServiceName returns the service name from telemetry settings (no-op for non-WASM)
func GetServiceName() string {
	return ""
}

// GetServiceVersion returns the service version from telemetry settings (no-op for non-WASM)
func GetServiceVersion() string {
	return ""
}

// GetResourceAttribute returns a specific resource attribute by key (no-op for non-WASM)
func GetResourceAttribute(key string) interface{} {
	return nil
}

// GetResourceAttributeString returns a resource attribute as a string (no-op for non-WASM)
func GetResourceAttributeString(key string) string {
	return ""
}

// GetAllResourceAttributes returns all resource attributes (no-op for non-WASM)
func GetAllResourceAttributes() map[string]interface{} {
	return make(map[string]interface{})
}