# Some rules are copied from https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension/blob/main/Makefile
# Some rules are copied from https://github.com/open-telemetry/opentelemetry-collector/blob/main/Makefile

gofumpt       := mvdan.cc/gofumpt@v0.5.0
gosimports    := github.com/rinchsan/gosimports/cmd/gosimports@v0.3.8
golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
wasibuilder   := github.com/tsuzu/wasibuilder@v0.0.1

# Function to execute a command. Note the empty line before endef to make sure each command
# gets executed separately instead of concatenated with previous one.
# Accepts command to execute as first parameter.
define exec-command
$(1)

endef

.PHONY: format
format:
	@go run $(gofumpt) -l -w .
	@go run $(gosimports) -w $(shell find . -name '*.go' -type f)

.PHONY: test
test:
	@(cd wasmprocessor; go test -v -tags docker ./...)
	@(cd wasmexporter; go test -v -tags docker ./...)
	@(cd wasmreceiver; go test -v -tags docker ./...)
	@(cd guest; go test -v -tags docker ./...)

examples/processor/nop/main.wasm: examples/processor/nop/main.go
	@(cd $(@D); GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./...)

examples/processor/add_new_attribute/main.wasm: examples/processor/add_new_attribute/main.go
	@(cd $(@D); GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./...)

examples/processor/curl/main.wasm: examples/processor/curl/main.go
	# getaddrinfo buildtag is necessary to use sock_getaddrinfo for name resolution
	@(cd $(@D); GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -tags="getaddrinfo" -o main.wasm ./...)

examples/exporter/nop/main.wasm: examples/exporter/nop/main.go
	@(cd $(@D); GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./...)

examples/exporter/stdout/main.wasm: examples/exporter/stdout/main.go
	@(cd $(@D); GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./...)

examples/receiver/nop/main.wasm: examples/receiver/nop/main.go
	@(cd $(@D); GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./...)

examples/receiver/webhookeventreceiver/main.wasm: examples/receiver/webhookeventreceiver/main.go
	# getaddrinfo buildtag is necessary to use sock_getaddrinfo for name resolution
	@(cd $(@D); GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -tags="getaddrinfo" -o main.wasm main.go)

examples/receiver/awss3receiver/main.wasm: examples/receiver/awss3receiver/main.go
	# getaddrinfo buildtag is necessary to use sock_getaddrinfo for name resolution
	@(cd $(@D); go run $(wasibuilder) go build -buildmode=c-shared -o main.wasm main.go)

.PHONY: build-wasm-examples
build-wasm-examples: examples/processor/nop/main.wasm examples/processor/add_new_attribute/main.wasm examples/processor/curl/main.wasm examples/exporter/nop/main.wasm examples/exporter/stdout/main.wasm examples/receiver/nop/main.wasm examples/receiver/webhookeventreceiver/main.wasm examples/receiver/awss3receiver/main.wasm

.PHONY: copy-wasm-examples
copy-wasm-examples: build-wasm-examples
	@mkdir -p wasmprocessor/testdata/nop wasmprocessor/testdata/add_new_attribute wasmprocessor/testdata/curl wasmexporter/testdata/nop wasmexporter/testdata/stdout wasmreceiver/testdata/nop
	@cp examples/processor/nop/main.wasm wasmprocessor/testdata/nop/
	@cp examples/processor/add_new_attribute/main.wasm wasmprocessor/testdata/add_new_attribute/
	@cp examples/processor/curl/main.wasm wasmprocessor/testdata/curl/
	@cp examples/exporter/nop/main.wasm wasmexporter/testdata/nop/
	@cp examples/exporter/stdout/main.wasm wasmexporter/testdata/stdout/
	@cp examples/receiver/nop/main.wasm wasmreceiver/testdata/nop/
	@cp examples/receiver/awss3receiver/main.wasm wasmreceiver/testdata/awss3/
