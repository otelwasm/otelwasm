# Some rules are copied from https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension/blob/main/Makefile
# Some rules are copied from https://github.com/open-telemetry/opentelemetry-collector/blob/main/Makefile

golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
wasibuilder   := github.com/otelwasm/wasibuilder@v0.0.6

SRC_ROOT := $(git rev-parse --show-toplevel)

GOCMD?= go
GO_BUILD_TAGS=""
GOARCH=$(shell $(GOCMD) env GOARCH)
GOOS=$(shell $(GOCMD) env GOOS)
GOTOOL := GOOS="" GOARCH="" $(GOCMD) tool

BUILDER := $(GOTOOL) builder

# Function to execute a command. Note the empty line before endef to make sure each command
# gets executed separately instead of concatenated with previous one.
# Accepts command to execute as first parameter.
define exec-command
$(1)

endef

.PHONY: format
format:
	@$(GOTOOL) gofumpt -l -w .
	@$(GOTOOL) gosimports -w $(shell find . -name '*.go' -type f)

.PHONY: test
test: copy-wasm-examples
	@(cd wasmprocessor; $(GOCMD) test -v -tags docker ./...)
	@(cd wasmexporter; $(GOCMD) test -v -tags docker ./...)
	@(cd wasmreceiver; $(GOCMD) test -v -tags docker ./...)
	@(cd guest; $(GOCMD) test -v -tags docker ./...)

.PHONY: examples/processor/nop/main.wasm
examples/processor/nop/main.wasm: examples/processor/nop/main.go
	@(cd $(@D); $(GOCMD) run $(wasibuilder) $(GOCMD) build -buildmode=c-shared -o main.wasm ./...)

.PHONY: examples/processor/add_new_attribute/main.wasm
examples/processor/add_new_attribute/main.wasm: examples/processor/add_new_attribute/main.go
	@(cd $(@D); $(GOCMD) run $(wasibuilder) $(GOCMD) build -buildmode=c-shared -o main.wasm ./...)

.PHONY: examples/processor/curl/main.wasm
examples/processor/curl/main.wasm: examples/processor/curl/main.go
	@(cd $(@D); $(GOCMD) run $(wasibuilder) $(GOCMD) build -buildmode=c-shared -o main.wasm ./...)

.PHONY: examples/exporter/nop/main.wasm
examples/exporter/nop/main.wasm: examples/exporter/nop/main.go
	@(cd $(@D); $(GOCMD) run $(wasibuilder) $(GOCMD) build -buildmode=c-shared -o main.wasm ./...)

.PHONY: examples/exporter/stdout/main.wasm
examples/exporter/stdout/main.wasm: examples/exporter/stdout/main.go
	@(cd $(@D); $(GOCMD) run $(wasibuilder) $(GOCMD) build -buildmode=c-shared -o main.wasm ./...)

.PHONY: examples/receiver/nop/main.wasm
examples/receiver/nop/main.wasm: examples/receiver/nop/main.go
	@(cd $(@D); $(GOCMD) run $(wasibuilder) $(GOCMD) build -buildmode=c-shared -o main.wasm ./...)

.PHONY: examples/receiver/webhookeventreceiver/main.wasm
examples/receiver/webhookeventreceiver/main.wasm: examples/receiver/webhookeventreceiver/main.go
	@(cd $(@D); $(GOCMD) run $(wasibuilder) $(GOCMD) build -buildmode=c-shared -o main.wasm main.go)

.PHONY: examples/receiver/awss3receiver/main.wasm
examples/receiver/awss3receiver/main.wasm: examples/receiver/awss3receiver/main.go
	@(cd $(@D); $(GOCMD) run $(wasibuilder) $(GOCMD) build -buildmode=c-shared -o main.wasm main.go)

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

.PHONY: docker-otelwasmcol
docker-otelwasmcol:
	GOOS=linux GOARCH=$(GOARCH) $(MAKE) otelwasmcol
	cp ./bin/otelwasmcol_linux_$(GOARCH) ./cmd/otelwasmcol/otelwasmcol
	docker build -t otelwasmcol ./cmd/otelwasmcol/
	rm ./cmd/otelwasmcol/otelwasmcol

.PHONY: genotelwasmcol
genotelwasmcol:
	$(BUILDER) --skip-compilation --config cmd/otelwasmcol/builder-config.yaml

.PHONY: otelwasmcol
otelwasmcol: genotelwasmcol
	cd ./cmd/otelwasmcol && GO111MODULE=on CGO_ENABLED=0 $(GOCMD) build -trimpath -o ../../bin/otelwasmcol_$(GOOS)_$(GOARCH)$(EXTENSION) \
		-tags $(GO_BUILD_TAGS) .

