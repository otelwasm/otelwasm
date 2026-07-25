# Some rules are copied from https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension/blob/main/Makefile
# Some rules are copied from https://github.com/open-telemetry/opentelemetry-collector/blob/main/Makefile

golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
wasibuilder   := github.com/otelwasm/wasibuilder@v0.0.6

SRC_ROOT := $(shell git rev-parse --show-toplevel)

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

# --- WasmEdge runtime support ---
# Building and testing with the WasmEdge runtime requires the WasmEdge shared
# library. Install it with `make install-wasmedge` (installs to ~/.wasmedge).
WASMEDGE_VERSION ?= 0.14.1
WASMEDGE_DIR ?= $(HOME)/.wasmedge
WASMEDGE_CGO_ENV := CGO_ENABLED=1 CGO_CFLAGS="-I$(WASMEDGE_DIR)/include" CGO_LDFLAGS="-L$(WASMEDGE_DIR)/lib -lwasmedge -Wl,-rpath,$(WASMEDGE_DIR)/lib"

.PHONY: install-wasmedge
install-wasmedge:
	curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install_v2.sh | bash -s -- -v $(WASMEDGE_VERSION)

.PHONY: test-wasmedge
test-wasmedge: copy-wasm-examples
	@(cd wasmplugin; $(WASMEDGE_CGO_ENV) $(GOCMD) test -v -tags wasmedge ./...)
	@(cd wasmprocessor; $(WASMEDGE_CGO_ENV) $(GOCMD) test -v -tags wasmedge ./...)
	@(cd wasmexporter; $(WASMEDGE_CGO_ENV) $(GOCMD) test -v -tags wasmedge ./...)
	@(cd wasmreceiver; $(WASMEDGE_CGO_ENV) $(GOCMD) test -v -tags wasmedge ./...)

# Precompiled AOT guests used by the WasmEdge AOT benchmarks
WASMEDGE_AOT_GUESTS := \
	wasmprocessor/testdata/attributesprocessor/main-aot.wasm \
	wasmprocessor/testdata/logs_add_attribute_handle/main-aot.wasm \
	wasmprocessor/testdata/logs_add_attribute_proto/main-aot.wasm \
	wasmprocessor/testdata/logs_add_attribute_handle_tinygo/main-aot.wasm \
	wasmprocessor/testdata/logs_add_attribute_handle_rust/main-aot.wasm

# Handle-ABI guests implemented with other toolchains for the guest-language
# benchmarks. Requires TinyGo and Rust (wasm32-wasip1 target).
.PHONY: build-wasm-guest-flavors
build-wasm-guest-flavors:
	(cd examples/processor/logs_add_attribute_handle_tinygo; \
		tinygo build -target=wasip1 -buildmode=c-shared -scheduler=none -gc=conservative -no-debug -o main.wasm .)
	(cd examples/processor/logs_add_attribute_handle_rust; \
		cargo build --release --target wasm32-wasip1 && \
		cp target/wasm32-wasip1/release/logs_add_attribute_handle_rust.wasm main.wasm)
	@for d in logs_add_attribute_handle_tinygo logs_add_attribute_handle_rust; do \
		mkdir -p wasmprocessor/testdata/$$d; \
		cp examples/processor/$$d/main.wasm wasmprocessor/testdata/$$d/; \
	done

wasmprocessor/testdata/%/main-aot.wasm: wasmprocessor/testdata/%/main.wasm
	$(WASMEDGE_DIR)/bin/wasmedge compile --optimize 2 $< $@

.PHONY: benchmark-wasmedge
benchmark-wasmedge: copy-wasm-examples $(WASMEDGE_AOT_GUESTS)
	@(cd wasmprocessor; $(WASMEDGE_CGO_ENV) $(GOCMD) test -run='^$$' -bench=. -benchmem -tags wasmedge)

define build-wasm-example
$(1): $(1:.wasm=.go)
	@(cd $$(patsubst %/main.wasm,%,$$@); $(GOCMD) run $(wasibuilder) $(GOCMD) build -buildmode=c-shared -o main.wasm main.go)
endef

# Automatically find all examples and generate rules based on directory structure.
# Guests built with other toolchains (TinyGo / Rust) are excluded here; see
# build-wasm-guest-flavors.
TELEMETRY_TYPES := processor exporter receiver
WASM_EXAMPLE_BINS := $(filter-out %_tinygo/main.wasm %_rust/main.wasm,$(foreach type,$(TELEMETRY_TYPES),$(patsubst examples/$(type)/%/main.go,examples/$(type)/%/main.wasm,$(shell find examples/$(type) -name 'main.go'))))

$(foreach example,$(WASM_EXAMPLE_BINS),$(eval $(call build-wasm-example,$(example))))

.PHONY: build-wasm-examples
build-wasm-examples: $(WASM_EXAMPLE_BINS)

.PHONY: copy-wasm-examples
copy-wasm-examples: build-wasm-examples
	@for example in $(WASM_EXAMPLE_BINS); do \
		type=$$(echo $$example | cut -d'/' -f2); \
		name=$$(echo $$example | cut -d'/' -f3); \
		target_dir="wasm$${type}/testdata/$${name}"; \
		mkdir -p "$${target_dir}"; \
		cp "$$example" "$${target_dir}/"; \
	done

.PHONY: benchmark
benchmark: copy-wasm-examples
	@echo "Running benchmarks for all modules..."
	@(cd wasmprocessor; $(GOCMD) test -run='^$$' -bench=. -benchmem -tags docker)
	@(cd wasmexporter; $(GOCMD) test -run='^$$' -bench=. -benchmem -tags docker)
	@(cd wasmreceiver; $(GOCMD) test -run='^$$' -bench=. -benchmem -tags docker)
	@(cd guest; $(GOCMD) test -run='^$$' -bench=. -benchmem -tags docker ./...)
	@echo "Benchmarks completed."

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

.PHONY: otelwasmcol-wasmedge
otelwasmcol-wasmedge: genotelwasmcol
	cd ./cmd/otelwasmcol && GO111MODULE=on $(WASMEDGE_CGO_ENV) $(GOCMD) build -trimpath -o ../../bin/otelwasmcol_$(GOOS)_$(GOARCH)$(EXTENSION) \
		-tags wasmedge .

.PHONY: factorybuilder
factorybuilder:
	$(GOCMD) build -o bin/factorybuilder ./cmd/factorybuilder

.PHONY: clean
clean:
	@find ./cmd/otelwasmcol -type f ! -name 'Dockerfile' ! -name '.gitignore' ! -name 'builder-config.yaml' -delete
