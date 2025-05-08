package processor

import (
	"go.opentelemetry.io/collector/component"
)

// Component metadata constants
var (
	Type             = component.MustNewType("add_new_attribute")
	TracesStability  = component.StabilityLevelAlpha
	MetricsStability = component.StabilityLevelAlpha
	LogsStability    = component.StabilityLevelAlpha
)
