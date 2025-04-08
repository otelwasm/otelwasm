module github.com/musaprg/otelwasm/pdata/proto

go 1.24.0

require (
	go.opentelemetry.io/proto/otlp v1.5.0
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/knqyf263/go-plugin v0.9.0 // indirect
	github.com/planetscale/vtprotobuf v0.4.0 // indirect
)

tool (
	github.com/knqyf263/go-plugin/cmd/protoc-gen-go-plugin
	google.golang.org/protobuf/cmd/protoc-gen-go
)
