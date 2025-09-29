package wazero

import (
	"github.com/otelwasm/otelwasm/runtime"
)

func init() {
	runtime.Register("wazero", newWazeroRuntime)
}
