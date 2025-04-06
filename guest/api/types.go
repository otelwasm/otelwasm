package api

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type Plugin interface{}

type TracesProcessor interface {
	Plugin

	ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *Status)
}
