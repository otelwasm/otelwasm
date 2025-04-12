package imports

import (
	"encoding/json"

	"github.com/musaprg/otelwasm/guest/internal/mem"
)

func GetConfig(v any) error {
	rawMsg := mem.GetBytes(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return getPluginConfig(ptr, limit)
	})
	return json.Unmarshal(rawMsg, v)
}
