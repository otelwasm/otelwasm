# Some rules are copied from https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension/blob/main/Makefile
# Some rules are copied from https://github.com/open-telemetry/opentelemetry-collector/blob/main/Makefile

gofumpt       := mvdan.cc/gofumpt@v0.5.0
gosimports    := github.com/rinchsan/gosimports/cmd/gosimports@v0.3.8
golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0

# Function to execute a command. Note the empty line before endef to make sure each command
# gets executed separately instead of concatenated with previous one.
# Accepts command to execute as first parameter.
define exec-command
$(1)

endef

.PHONY: proto-tools
proto-tools:
	cd ./pdata/proto/tools; \
	cat tools.go | grep "_" | awk -F'"' '{print $$2}' | xargs -tI % go install %

# TODO: Add rule to install protoc command as a dependency

OPENTELEMETRY_PROTO_VERSION=v1.5.0
.PHONY: submodule-update
submodule-update:
	git submodule update -i
	cd ./pdata/opentelemetry-proto; \
	git checkout $(OPENTELEMETRY_PROTO_VERSION)
	# TODO: consider sparse checkout

OPENTELEMETRY_PROTO_SRC_DIR=pdata/opentelemetry-proto
OPENTELEMETRY_PROTO_FILES := $(subst $(OPENTELEMETRY_PROTO_SRC_DIR)/,,$(wildcard $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/*/v1/*.proto $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/*/v1development/*.proto))

# This uses the exact generated protos from Kubernetes source, to ensure exact
# wire-type parity. Otherwise, we need expensive to maintain conversion logic.
# We can't use the go generated in the same source tree in TinyGo, because it
# hangs compiling. Instead, we generate UnmarshalVT with go-plugin which is
# known to work with TinyGo.
.PHONY: update-pdata-proto
update-pdata-proto: proto-tools
	@echo Generating code for the following files:
	@$(foreach file,$(OPENTELEMETRY_PROTO_FILES),$(call exec-command,echo $(file)))
	cd ./pdata/opentelemetry-proto; \
	protoc ./opentelemetry/proto/common/v1/common.proto --go-plugin_out=../proto \
		--go-plugin_opt=Mopentelemetry/proto/common/v1/common.proto=./pcommon; \
	protoc ./opentelemetry/proto/logs/v1/logs.proto --go-plugin_out=../proto \
		--go-plugin_opt=Mopentelemetry/proto/logs/v1/logs.proto=./plog; \
	protoc ./opentelemetry/proto/metrics/v1/metrics.proto --go-plugin_out=../proto \
		--go-plugin_opt=Mopentelemetry/proto/metrics/v1/metrics.proto=./pmetric; \
	protoc ./opentelemetry/proto/trace/v1/trace.proto --go-plugin_out=../proto \
		--go-plugin_opt=Mopentelemetry/proto/trace/v1/trace.proto=./ptrace; \
	# @$(MAKE) format

.PHONY: format
format:
	@go run $(gofumpt) -l -w .
	@go run $(gosimports) -local sigs.k8s.io/kube-scheduler-wasm-extension/ -w $(shell find . -name '*.go' -type f)

.PHONY: test
test:
	$(cd wasmreceiver; go test -v ./...)
	@(cd wasmprocessor; go test -v ./...)
	@(cd wasmexporter; go test -v ./...)
	@(cd guest; go test -v ./...)

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

.PHONY: build-wasm-examples
build-wasm-examples: examples/processor/nop/main.wasm examples/processor/add_new_attribute/main.wasm examples/processor/curl/main.wasm examples/exporter/nop/main.wasm examples/exporter/stdout/main.wasm examples/receiver/nop/main.wasm

.PHONY: copy-wasm-examples
copy-wasm-examples: build-wasm-examples
	@mkdir -p wasmprocessor/testdata/nop wasmprocessor/testdata/add_new_attribute wasmprocessor/testdata/curl wasmexporter/testdata/nop wasmexporter/testdata/stdout wasmreceiver/testdata/nop
	@cp examples/processor/nop/main.wasm wasmprocessor/testdata/nop/
	@cp examples/processor/add_new_attribute/main.wasm wasmprocessor/testdata/add_new_attribute/
	@cp examples/processor/curl/main.wasm wasmprocessor/testdata/curl/
	@cp examples/exporter/nop/main.wasm wasmexporter/testdata/nop/
	@cp examples/exporter/stdout/main.wasm wasmexporter/testdata/stdout/
	@cp examples/receiver/nop/main.wasm wasmreceiver/testdata/nop/
