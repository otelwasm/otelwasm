module github.com/otelwasm/otelwasm/cmd/wasmpush

go 1.24.2

replace github.com/otelwasm/otelwasm/wasmplugin => ../../wasmplugin

require github.com/otelwasm/otelwasm/wasmplugin v0.0.0-00010101000000-000000000000

require (
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	golang.org/x/sync v0.13.0 // indirect
	oras.land/oras-go/v2 v2.5.0 // indirect
)
