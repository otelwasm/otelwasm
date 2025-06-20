//go:build !wasm

package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetrySettings(t *testing.T) {
	// Test that TelemetrySettings struct can be created
	settings := TelemetrySettings{
		ResourceAttributes: map[string]interface{}{
			"service.name":    "test-service",
			"service.version": "1.0.0",
		},
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ComponentID:    map[string]string{"type": "processor"},
	}

	assert.Equal(t, "test-service", settings.ServiceName)
	assert.Equal(t, "1.0.0", settings.ServiceVersion)
	assert.Equal(t, "test-service", settings.ResourceAttributes["service.name"])
	assert.Equal(t, "processor", settings.ComponentID["type"])
}

func TestGetTelemetrySettings(t *testing.T) {
	// For non-WASM builds, this should return empty settings
	settings, err := GetTelemetrySettings()
	require.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Empty(t, settings.ServiceName)
	assert.Empty(t, settings.ServiceVersion)
	assert.Empty(t, settings.ResourceAttributes)
	assert.Empty(t, settings.ComponentID)
}

func TestGetServiceName(t *testing.T) {
	// For non-WASM builds, this should return empty string
	name := GetServiceName()
	assert.Empty(t, name)
}

func TestGetServiceVersion(t *testing.T) {
	// For non-WASM builds, this should return empty string
	version := GetServiceVersion()
	assert.Empty(t, version)
}

func TestGetResourceAttribute(t *testing.T) {
	// For non-WASM builds, this should return nil
	attr := GetResourceAttribute("any.key")
	assert.Nil(t, attr)
}

func TestGetResourceAttributeString(t *testing.T) {
	// For non-WASM builds, this should return empty string
	attr := GetResourceAttributeString("any.key")
	assert.Empty(t, attr)
}

func TestGetAllResourceAttributes(t *testing.T) {
	// For non-WASM builds, this should return empty map
	attrs := GetAllResourceAttributes()
	assert.NotNil(t, attrs)
	assert.Empty(t, attrs)
}