package wazero

import (
	"github.com/otelwasm/otelwasm/runtime"
	"github.com/otelwasm/otelwasm/wasmplugin"
)

func init() {
	runtime.Register(wasmplugin.RuntimeTypeWazero, newWazeroRuntime)
}