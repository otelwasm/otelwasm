//go:build wasm

package telemetry

import (
	"encoding/json"

	"github.com/otelwasm/otelwasm/guest/internal/mem"
)

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

//go:wasmimport opentelemetry.io/wasm getTelemetrySettings
func getTelemetrySettings(ptr uint32, limit mem.BufLimit) (len uint32)

// GetTelemetrySettings retrieves telemetry settings from the host
func GetTelemetrySettings() (*TelemetrySettings, error) {
	rawMsg := mem.GetBytes(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return getTelemetrySettings(ptr, limit)
	})

	var settings TelemetrySettings
	if err := json.Unmarshal(rawMsg, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// GetServiceName returns the service name from telemetry settings
func GetServiceName() string {
	settings, err := GetTelemetrySettings()
	if err != nil {
		return ""
	}
	return settings.ServiceName
}

// GetServiceVersion returns the service version from telemetry settings
func GetServiceVersion() string {
	settings, err := GetTelemetrySettings()
	if err != nil {
		return ""
	}
	return settings.ServiceVersion
}

// GetResourceAttribute returns a specific resource attribute by key
func GetResourceAttribute(key string) interface{} {
	settings, err := GetTelemetrySettings()
	if err != nil {
		return nil
	}
	return settings.ResourceAttributes[key]
}

// GetResourceAttributeString returns a resource attribute as a string
func GetResourceAttributeString(key string) string {
	value := GetResourceAttribute(key)
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

// GetAllResourceAttributes returns all resource attributes
func GetAllResourceAttributes() map[string]interface{} {
	settings, err := GetTelemetrySettings()
	if err != nil {
		return make(map[string]interface{})
	}
	return settings.ResourceAttributes
}