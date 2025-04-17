package wasmexporter

import "github.com/musaprg/otelwasm/wasmplugin"

type Config struct {
	wasmplugin.Config `mapstructure:",squash"`
}

func (cfg *Config) Validate() error {
	return cfg.Config.Validate()
}
