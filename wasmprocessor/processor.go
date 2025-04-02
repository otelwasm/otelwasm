package wasmprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

type wasmProcessor struct {
}

func (wp *wasmProcessor) processTraces(
	ctx context.Context,
	td ptrace.Traces,
) (ptrace.Traces, error) {
	return td, nil
}
