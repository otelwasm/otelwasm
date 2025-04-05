package wasmprocessor

import "fmt"

type Config struct {
	Path string `mapstructure:"path"`
}

func (cfg *Config) Validate() error {
	if cfg.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}
