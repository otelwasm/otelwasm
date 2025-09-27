# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OTelWasm is an experimental WebAssembly-based OpenTelemetry Collector plugins system. It extends the OpenTelemetry Collector with WASM-powered components for receivers, processors, and exporters.

## Development Commands

### Building
- `make otelwasmcol` - Build the OTelWasm OTel Collector distribution (binary in `bin/`)
- `make build-wasm-examples` - Build all example WASM components 
- `make copy-wasm-examples` - Build examples and copy to testdata directories
- `make docker-otelwasmcol` - Build Docker image of otelwasmcol
- `make factorybuilder` - Build the factory builder tool

### Testing
- `make test` - Run tests for all modules (requires Docker, builds WASM examples first)
- `cd <module> && go test -tags docker -v ./...` - Run tests for specific module
- `make benchmark` - Run benchmarks for all modules

### Code Quality
- `make format` - Format code using gofumpt and gosimports
- `make clean` - Remove generated files from cmd/otelwasmcol

### Individual Module Testing
Each major component has its own go.mod:
- `wasmprocessor/` - WASM processor component
- `wasmexporter/` - WASM exporter component  
- `wasmreceiver/` - WASM receiver component
- `guest/` - Guest-side WASM plugin framework
- `wasmplugin/` - Core WASM plugin runtime

## Architecture

### Multi-Module Structure
The project uses a multi-module Go workspace with separate go.mod files for each component. The root go.mod is only for tools management.

### Core Components
- **wasmplugin/** - Core WASM runtime using wazero interpreter (not compiler, see RATIONALE.md)
- **guest/** - Guest-side SDK for building WASM plugins, provides OpenTelemetry interfaces
- **wasm{processor,exporter,receiver}/** - Host-side components that load and execute WASM plugins
- **examples/** - Example WASM implementations for each component type
- **cmd/factorybuilder/** - Code generator for OpenTelemetry factory boilerplate
- **cmd/otelwasmcol/** - Main collector binary with WASM components

### WASM Plugin System
- Guest plugins are built using `wasibuilder` tool with `-buildmode=c-shared`
- Host runtime uses wazero interpreter (not compiler runtime due to stability issues)
- Plugins implement standard OpenTelemetry interfaces (receiver, processor, exporter)
- Communication uses host function imports for memory management and telemetry data

### Build Process
1. WASM examples built with `wasibuilder go build -buildmode=c-shared`
2. Built WASM files copied to component testdata directories
3. OTel Collector built using `go.opentelemetry.io/collector/cmd/builder`
4. Final binary combines core OTel components with WASM-enabled components

## Testing Requirements

All tests require Docker and the `docker` build tag. Tests automatically build required WASM examples before running.

## Configuration Example

WASM components are configured in collector config with `wasm/` prefix:
```yaml
processors:
  wasm/attributes:
    path: "./examples/processor/attributesprocessor/main.wasm"
    plugin_config:
      # Standard OpenTelemetry component config
```

## Known Limitations

- OTLP/gRPC not supported in receivers (use OTLP/HTTP)
- Compression and sending_queue must be disabled in exporters
- Project is experimental, not for production use
