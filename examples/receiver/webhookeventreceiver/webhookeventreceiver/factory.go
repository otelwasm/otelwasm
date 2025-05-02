// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package webhookeventreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/webhookeventreceiver"

import (
	"github.com/otelwasm/otelwasm/examples/receiver/webhookeventreceiver/webhookeventreceiver/metadata"
	"go.opentelemetry.io/collector/component"
)

var scopeLogName = "otlp/" + metadata.Type.String()

const (
	// might add this later, for now I wish to require a valid
	// endpoint to be declared by the user.
	// Default endpoints to bind to.
	// defaultEndpoint = "localhost:8080"
	defaultReadTimeout  = "500ms"
	defaultWriteTimeout = "500ms"
	defaultPath         = "/events"
	defaultHealthPath   = "/health_check"
)

// Default configuration for the generic webhook receiver
func CreateDefaultConfig() component.Config {
	return &Config{
		Path:                       defaultPath,
		HealthPath:                 defaultHealthPath,
		ReadTimeout:                defaultReadTimeout,
		WriteTimeout:               defaultWriteTimeout,
		ConvertHeadersToAttributes: false, // optional, off by default
		SplitLogsAtNewLine:         false,
	}
}
