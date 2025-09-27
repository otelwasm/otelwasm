# OTelWasm Application Binary Interface (ABI) Documentation

## Overview

OTelWasm implements a WebAssembly-based plugin system for OpenTelemetry Collector components. This document describes the ABI (Application Binary Interface) between the host (OTel Collector) and guest (WASM plugins) for each component type.

## Architecture

The ABI is built on several key concepts:

1. **Host Module**: `opentelemetry.io/wasm` - provides functions to guests
2. **Guest Exports**: Functions exported by WASM modules that the host calls
3. **Memory Management**: Shared memory space for data exchange
4. **Telemetry Data**: OpenTelemetry protocol buffers (protobuf) serialization
5. **Status Codes**: Return values indicating success/failure

## Common Host Functions

All component types have access to these host functions:

### Data Retrieval Functions

- **`currentTraces(buf: i32, buf_limit: i32) -> i32`**
  - Retrieves current trace data from host into guest buffer
  - Returns actual size written or required size if buffer too small

- **`currentMetrics(buf: i32, buf_limit: i32) -> i32`**
  - Retrieves current metrics data from host into guest buffer
  - Returns actual size written or required size if buffer too small

- **`currentLogs(buf: i32, buf_limit: i32) -> i32`**
  - Retrieves current logs data from host into guest buffer
  - Returns actual size written or required size if buffer too small

### Result Setting Functions

- **`setResultTraces(buf: i32, buf_len: i32)`**
  - Sends processed trace data back to host
  - Data must be protobuf-serialized ptrace.Traces

- **`setResultMetrics(buf: i32, buf_len: i32)`**
  - Sends processed metrics data back to host
  - Data must be protobuf-serialized pmetric.Metrics

- **`setResultLogs(buf: i32, buf_len: i32)`**
  - Sends processed logs data back to host
  - Data must be protobuf-serialized plog.Logs

### Configuration and Control Functions

- **`getPluginConfig(buf: i32, buf_limit: i32) -> i32`**
  - Retrieves plugin configuration as JSON
  - Configuration comes from collector's YAML config `plugin_config` section

- **`setResultStatusReason(buf: i32, buf_len: i32)`**
  - Sets error reason string for non-success status codes

- **`getShutdownRequested() -> i32`**
  - Returns 1 if shutdown requested, 0 otherwise
  - Used by receivers to gracefully shut down

## Common Guest Exports

### Required Export

- **`getSupportedTelemetry() -> u32`**
  - Returns bitfield indicating supported telemetry types
  - Bit 0 (1): Metrics support
  - Bit 1 (2): Logs support
  - Bit 2 (4): Traces support

## Component-Specific ABIs

### Processor Components

Processors transform telemetry data as it flows through the collector pipeline.

#### Guest Exports

##### Traces Processor
- **`processTraces() -> u32`**
  - Called when trace data needs processing
  - Returns status code (0 = success, 1 = error, 2 = invalid argument)
  - Must call `currentTraces()` to get input data
  - Must call `setResultTraces()` to return processed data

##### Metrics Processor
- **`processMetrics() -> u32`**
  - Called when metrics data needs processing
  - Returns status code
  - Must call `currentMetrics()` to get input data
  - Must call `setResultMetrics()` to return processed data

##### Logs Processor
- **`processLogs() -> u32`**
  - Called when logs data needs processing
  - Returns status code
  - Must call `currentLogs()` to get input data
  - Must call `setResultLogs()` to return processed data

#### Data Flow

1. Host calls `processTraces()`/`processMetrics()`/`processLogs()`
2. Guest calls `currentTraces()`/`currentMetrics()`/`currentLogs()` to get input data
3. Guest processes the data
4. Guest calls `setResultTraces()`/`setResultMetrics()`/`setResultLogs()` with results
5. Guest returns status code to host

#### Example Implementation

```go
//go:wasmexport processTraces
func _processTraces() uint32 {
    traces := imports.CurrentTraces()
    result, status := processor.ProcessTraces(traces)
    if result != (ptrace.Traces{}) {
        imports.SetResultTraces(result)
    }
    return imports.StatusToCode(status)
}
```

### Receiver Components

Receivers pull telemetry data from external sources and inject it into the collector pipeline.

#### Guest Exports

##### Traces Receiver
- **`startTracesReceiver()`**
  - Called to start the receiver
  - Should run continuously until shutdown requested
  - Must periodically check `getShutdownRequested()`
  - Calls `setResultTraces()` to send received data

##### Metrics Receiver
- **`startMetricsReceiver()`**
  - Called to start the receiver
  - Must periodically check `getShutdownRequested()`
  - Calls `setResultMetrics()` to send received data

##### Logs Receiver
- **`startLogsReceiver()`**
  - Called to start the receiver
  - Must periodically check `getShutdownRequested()`
  - Calls `setResultLogs()` to send received data

#### Data Flow

1. Host calls `startTracesReceiver()`/`startMetricsReceiver()`/`startLogsReceiver()`
2. Guest runs continuously, receiving data from external sources
3. Guest calls `setResultTraces()`/`setResultMetrics()`/`setResultLogs()` when data received
4. Guest checks `getShutdownRequested()` periodically and exits when requested

#### Example Implementation

```go
//go:wasmexport startTracesReceiver
func _startTracesReceiver() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go func() {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                if imports.GetShutdownRequested() {
                    cancel()
                    return
                }
            case <-ctx.Done():
                return
            }
        }
    }()

    receiver.StartTraces(ctx)
}
```

### Exporter Components

Exporters send telemetry data to external systems.

#### Guest Exports

##### Traces Exporter
- **`pushTraces() -> u32`**
  - Called when trace data needs to be exported
  - Returns status code
  - Must call `currentTraces()` to get data to export
  - Should send data to external system

##### Metrics Exporter
- **`pushMetrics() -> u32`**
  - Called when metrics data needs to be exported
  - Returns status code
  - Must call `currentMetrics()` to get data to export

##### Logs Exporter
- **`pushLogs() -> u32`**
  - Called when logs data needs to be exported
  - Returns status code
  - Must call `currentLogs()` to get data to export

#### Data Flow

1. Host calls `pushTraces()`/`pushMetrics()`/`pushLogs()`
2. Guest calls `currentTraces()`/`currentMetrics()`/`currentLogs()` to get data
3. Guest exports data to external system
4. Guest returns status code to host

#### Example Implementation

```go
//go:wasmexport pushTraces
func _pushTraces() uint32 {
    traces := imports.CurrentTraces()
    status := exporter.PushTraces(traces)
    return imports.StatusToCode(status)
}
```

## Status Codes

Status codes are returned as `uint32` values:

- **0**: `StatusCodeSuccess` - Operation completed successfully
- **1**: `StatusCodeError` - Operation failed with error
- **2**: `StatusCodeInvalidArgument` - Invalid input provided (defined in host but not widely used)

Error details can be provided using `setResultStatusReason()`.

## Data Serialization

All telemetry data exchanged between host and guest uses Protocol Buffer serialization:

- **Traces**: `ptrace.Traces` serialized with `ptrace.ProtoMarshaler`
- **Metrics**: `pmetric.Metrics` serialized with `pmetric.ProtoMarshaler`
- **Logs**: `plog.Logs` serialized with `plog.ProtoMarshaler`

Configuration data is exchanged as JSON.

## Memory Management

The ABI uses a shared memory model:

1. Guest exports a `memory` section accessible to host
2. Data is passed via memory pointers and lengths
3. Host functions write to guest memory buffers
4. Guest functions read from memory and write results back
5. Memory management is handled by guest-side helper functions in `guest/internal/mem`

## WASI Integration

The system uses WASI (WebAssembly System Interface) for:

- Standard I/O operations
- Network sockets (via WasmEdge v2 extension)
- Environment variables
- File system access (limited)

## Build Requirements

WASM plugins must be built with:

- **Build mode**: `c-shared` (using `wasibuilder`)
- **Target**: `wasm32-wasi`
- **Exports**: Required component-specific functions
- **Imports**: Host functions from `opentelemetry.io/wasm`

## Limitations

1. **OTLP/gRPC**: Currently not supported in receivers (use OTLP/HTTP)
2. **Compression**: Must be disabled in exporters
3. **Sending Queue**: Must be disabled in exporters
4. **Runtime**: Uses wazero interpreter mode (not compiler mode)

## Example Configuration

```yaml
processors:
  wasm/attributes:
    path: "./examples/processor/attributesprocessor/main.wasm"
    plugin_config:
      actions:
      - key: inserted-attributes-by-wasm
        action: insert
        value: hello-from-wasm

receivers:
  wasm/otlpreceiver:
    path: "./examples/receiver/otlpreceiver/main.wasm"
    plugin_config:
      protocols:
        http:
          endpoint: "0.0.0.0:4318"

exporters:
  wasm/otlphttpexporter:
    path: "./examples/exporter/otlphttpexporter/main.wasm"
    plugin_config:
      endpoint: "http://localhost:4319"
      compression: none
      sending_queue:
        enabled: false
```