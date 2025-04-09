package plugin

import (
	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/metricsprocessor"
	"github.com/musaprg/otelwasm/guest/tracesprocessor"
)

func Set(plugin api.Plugin) {
	if plugin, ok := plugin.(api.TracesProcessor); ok {
		tracesprocessor.SetPlugin(plugin)
	}
	if plugin, ok := plugin.(api.MetricsProcessor); ok {
		metricsprocessor.SetPlugin(plugin)
	}
}
