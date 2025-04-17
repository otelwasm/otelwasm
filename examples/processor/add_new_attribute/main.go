package main

import (
	"fmt"

	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/imports"
	"github.com/musaprg/otelwasm/guest/plugin" // register tracesprocessor
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func init() {
	plugin.Set(&AttributeProcessor{})
}
func main() {}

var _ api.TracesProcessor = (*AttributeProcessor)(nil)

type AttributeProcessor struct{}

type Config struct {
	AttributeName  string `json:"attribute_name"`
	AttributeValue string `json:"attribute_value"`
}

func (c *Config) Validate() error {
	if c.AttributeName == "" {
		return fmt.Errorf("attribute_name is required")
	}
	if c.AttributeValue == "" {
		return fmt.Errorf("attribute_value is required")
	}
	return nil
}

// ProcessTraces implements api.TracesProcessor.
func (n *AttributeProcessor) ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *api.Status) {
	// Get config
	config := &Config{}
	imports.GetConfig(config)

	fmt.Printf("Config loaded: %v\n", config)

	if err := config.Validate(); err != nil {
		return ptrace.Traces{}, &api.Status{
			Code:   api.StatusCodeError,
			Reason: err.Error(),
		}
	}

	fmt.Printf("Config validated: %v\n", config)

	// Add new attribute to all spans using config values
	newTraces := ptrace.NewTraces()
	traces.CopyTo(newTraces)
	rSpans := newTraces.ResourceSpans()
	for i := 0; i < rSpans.Len(); i++ {
		scopeSpans := rSpans.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				spans.At(k).Attributes().PutStr(config.AttributeName, config.AttributeValue)
			}
		}
	}

	fmt.Printf("New attribute added: %s=%s\n", config.AttributeName, config.AttributeValue)

	return newTraces, api.StatusSuccess()
}
