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

define build-wasm-example
$(1): $(1:.wasm=.go)
	@(cd $$(patsubst %/main.wasm,%,$$@); $(GOCMD) run $(wasibuilder) $(GOCMD) build -buildmode=c-shared -o main.wasm main.go)
endef

# Automatically find all examples and generate rules based on directory structure
TELEMETRY_TYPES := processor exporter receiver
WASM_EXAMPLE_BINS := $(foreach type,$(TELEMETRY_TYPES),$(patsubst examples/$(type)/%/main.go,examples/$(type)/%/main.wasm,$(shell find examples/$(type) -name 'main.go')))

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

.PHONY: factorybuilder
factorybuilder:
	$(GOCMD) build -o bin/factorybuilder ./cmd/factorybuilder

.PHONY: clean
clean:
	@find ./cmd/otelwasmcol -type f ! -name 'Dockerfile' ! -name '.gitignore' ! -name 'builder-config.yaml' -delete
