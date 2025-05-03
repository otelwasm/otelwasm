module github.com/otelwasm/otelwasm

// NOTE: This go.mod file is only used for tools management.
// It is not used for building the project.
// The actual go.mod files are in each submodule.

go 1.24.2

tool (
	github.com/rinchsan/gosimports/cmd/gosimports
	go.opentelemetry.io/collector/cmd/builder
	mvdan.cc/gofumpt
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/parsers/yaml v1.0.0 // indirect
	github.com/knadh/koanf/providers/env v1.1.0 // indirect
	github.com/knadh/koanf/providers/file v1.2.0 // indirect
	github.com/knadh/koanf/providers/fs v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.2.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/otelwasm/wasibuilder v0.0.6 // indirect
	github.com/rinchsan/gosimports v0.3.8 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	go.opentelemetry.io/collector/cmd/builder v0.125.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/tools v0.13.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mvdan.cc/gofumpt v0.5.0 // indirect
)
