package wasmreceiver

import "github.com/otelwasm/otelwasm/wasmplugin"

type Config struct {
	wasmplugin.Config `mapstructure:",squash"`
}

func (cfg *Config) Validate() error {
	return cfg.Config.Validate()
}
