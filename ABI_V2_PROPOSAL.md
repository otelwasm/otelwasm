# OTelWasm ABI v2 Design Proposal

## Executive Summary

Based on analysis of the current otelwasm ABI limitations and best practices from proxy-wasm and kube-scheduler-wasm-extension, this document proposes a comprehensive redesign for ABI v2 that addresses critical issues around memory management, error handling, streaming support, and extensibility.

## Current ABI Limitations

### Critical Issues
1. **Unsafe Memory Management**: Manual `buf/buf_limit` patterns prone to buffer overruns
2. **Limited Error Handling**: Only 3 status codes, separate error reason calls
3. **No Streaming Support**: Large datasets must be processed in single memory allocations
4. **Fixed Serialization**: Hardcoded to protobuf only
5. **Weak Lifecycle Management**: No init/configure phases, limited shutdown coordination
6. **Poor Extensibility**: Component-specific functions, no capability negotiation

### Known Bugs
- OTLP/gRPC not supported in receivers
- Compression must be disabled in exporters
- Sending queue must be disabled in exporters

## ABI v2 Design Principles

### 1. Safety First
- Guest-managed memory allocation
- Structured error handling with detailed reasons
- Strong typing through formal contracts

### 2. Performance & Scalability
- Streaming support for large datasets
- Lazy loading for complex data structures
- Efficient serialization negotiation

### 3. Extensibility
- Service-oriented interface design
- Capability negotiation mechanism
- Formal versioning system

### 4. Developer Experience
- Clear lifecycle hooks
- Rich debugging information
- Auto-generated language bindings

## Core ABI v2 Design

### Memory Management Model

**Principle**: Guest owns and manages all its memory; host never writes to arbitrary locations.

#### Guest Exports
```wasm
;; Allocate memory in guest heap, return pointer
(func (export "allocate") (param $size i32) (result i32))

;; Free previously allocated memory
(func (export "deallocate") (param $ptr i32) (param $size i32))
```

#### Host-Guest Data Flow
1. Host calls `guest.allocate(data_size)` → receives `guest_ptr`
2. Host writes data to guest memory at `guest_ptr`
3. Host calls processing function with `(guest_ptr, data_size)`
4. Guest processes data and calls `deallocate(guest_ptr, data_size)` when done

#### Benefits
- Eliminates buffer overflow risks
- Guest has full control over memory layout
- Simpler host implementation
- Follows proxy-wasm proven pattern

### Error Handling & Status System

**Principle**: Structured status with detailed error information.

#### Status Structure
```proto
message Status {
  StatusCode code = 1;
  string reason = 2;
  map<string, string> details = 3; // Additional debug info
}

enum StatusCode {
  OK = 0;
  ERROR = 1;
  INVALID_ARGUMENT = 2;
  UNSCHEDULABLE = 3;  // For receivers that can't process data
  SKIP = 4;           // For processors that skip certain data
  RETRY = 5;          // For temporary failures
}
```

#### Host Functions
```wasm
;; Get detailed status information after a call
(func (import "otelwasm.v2" "get_last_status")
  (param $buf_ptr i32) (param $buf_size i32) (result i32))
```

#### Benefits
- Rich error information for debugging
- Structured status codes for different scenarios
- Extensible details for component-specific information
- Follows kube-scheduler-wasm-extension pattern

### Streaming & Data Exchange

**Principle**: Support both batch and streaming processing for large datasets.

#### Streaming Interface
```wasm
;; Start a streaming data session
(func (import "otelwasm.v2" "stream_start")
  (param $data_type i32) (param $total_size i64) (result i32))

;; Get next chunk of streaming data
(func (import "otelwasm.v2" "stream_read_chunk")
  (param $buf_ptr i32) (param $buf_size i32) (result i32))

;; Check if more data available
(func (import "otelwasm.v2" "stream_has_more") (result i32))

;; End streaming session
(func (import "otelwasm.v2" "stream_end"))
```

#### Batch Interface (for backward compatibility)
```wasm
;; Process entire dataset in one call (for smaller datasets)
(func (export "process_data_batch")
  (param $data_type i32) (param $data_ptr i32) (param $data_size i32) (result i32))
```

#### Benefits
- Handles arbitrarily large datasets
- Memory-efficient processing
- Backward compatibility with batch processing
- Enables real-time streaming processors

### Lifecycle Management

**Principle**: Event-driven lifecycle with clear phases.

#### Guest Exports (Lifecycle)
```wasm
;; Plugin initialization - called once at startup
(func (export "on_plugin_start") (param $config_ptr i32) (param $config_size i32) (result i32))

;; Reconfiguration - called when config changes
(func (export "on_plugin_configure") (param $config_ptr i32) (param $config_size i32) (result i32))

;; Periodic tick - called at regular intervals
(func (export "on_plugin_tick") (result i32))

;; Health check - called to verify plugin health
(func (export "on_plugin_health_check") (result i32))

;; Graceful shutdown - called before termination
(func (export "on_plugin_shutdown") (result i32))
```

#### Benefits
- Clear initialization and cleanup phases
- Support for dynamic reconfiguration
- Health monitoring capabilities
- Graceful shutdown coordination

### Capability Negotiation & Versioning

**Principle**: Formal ABI versioning with capability discovery.

#### Capability Structure
```proto
message PluginCapabilities {
  uint32 abi_version = 1;
  repeated TelemetryType supported_telemetry = 2;
  repeated SerializationFormat supported_formats = 3;
  bool supports_streaming = 4;
  bool supports_batch_processing = 5;
  bool supports_lazy_loading = 6;
  map<string, bool> extensions = 7;
}

enum TelemetryType {
  TRACES = 0;
  METRICS = 1;
  LOGS = 2;
  PROFILES = 3;  // Future extension
}

enum SerializationFormat {
  OTLP_PROTOBUF = 0;
  OTLP_JSON = 1;
  CUSTOM = 2;
}
```

#### Guest Exports (Capability)
```wasm
;; Return plugin capabilities and supported ABI version
(func (export "get_plugin_capabilities") (param $buf_ptr i32) (param $buf_size i32) (result i32))
```

#### Benefits
- Prevents runtime compatibility issues
- Enables gradual feature rollouts
- Future-proofs the ABI
- Allows performance optimizations based on capabilities

### Service-Oriented Interface

**Principle**: Component interfaces defined as services with clear contracts.

#### Component Services (Guest Exports)
```wasm
;; Processor Service
(func (export "traces_processor.process")
  (param $input_ptr i32) (param $input_size i32)
  (param $output_ptr i32) (param $output_size i32) (result i32))

(func (export "metrics_processor.process")
  (param $input_ptr i32) (param $input_size i32)
  (param $output_ptr i32) (param $output_size i32) (result i32))

;; Receiver Service
(func (export "traces_receiver.start") (result i32))
(func (export "traces_receiver.stop") (result i32))

;; Exporter Service
(func (export "traces_exporter.export")
  (param $data_ptr i32) (param $data_size i32) (result i32))
```

#### Host Services (Host Exports)
```wasm
;; Configuration Service
(func (import "otelwasm.v2.config" "get_config")
  (param $key_ptr i32) (param $key_size i32)
  (param $buf_ptr i32) (param $buf_size i32) (result i32))

;; Observability Service
(func (import "otelwasm.v2.observability" "emit_metric")
  (param $metric_ptr i32) (param $metric_size i32) (result i32))

(func (import "otelwasm.v2.observability" "log_message")
  (param $level i32) (param $msg_ptr i32) (param $msg_size i32) (result i32))

;; Data Service
(func (import "otelwasm.v2.data" "send_traces")
  (param $traces_ptr i32) (param $traces_size i32) (result i32))
```

#### Benefits
- Clear separation of concerns
- Extensible service architecture
- Auto-generated client libraries
- Consistent interface patterns

### Formal ABI Definition

**Principle**: Machine-readable ABI contract definition.

#### Option 1: Protocol Buffers
```proto
// otelwasm_v2.proto
service TracesProcessor {
  rpc Process(ProcessTracesRequest) returns (ProcessTracesResponse);
}

message ProcessTracesRequest {
  bytes otlp_data = 1;
  SerializationFormat format = 2;
}

message ProcessTracesResponse {
  bytes otlp_data = 1;
  Status status = 2;
}
```

#### Option 2: WebAssembly Interface Types (WIT)
```wit
// otelwasm-v2.wit
interface traces-processor {
  process: func(data: list<u8>, format: serialization-format) -> result<list<u8>, status>
}

record status {
  code: status-code,
  reason: string,
  details: list<tuple<string, string>>
}
```

#### Benefits
- Machine-readable contracts
- Auto-generated bindings for multiple languages
- Version compatibility checking
- Clear documentation from interface definition

## Component-Specific ABI Details

### Processors

#### Interface
```wasm
;; Functional processing with explicit input/output
(func (export "traces_processor.process")
  (param $input_ptr i32) (param $input_size i32)
  (param $output_ptr i32) (param $output_size i32) (result i32))

;; Streaming processing for large datasets
(func (export "traces_processor.process_stream") (result i32))
```

#### Data Flow
1. Host calls `guest.allocate(input_size)` → gets `input_ptr`
2. Host writes input data to `input_ptr`
3. Host calls `guest.allocate(max_output_size)` → gets `output_ptr`
4. Host calls `traces_processor.process(input_ptr, input_size, output_ptr, max_output_size)`
5. Guest processes data, writes result to `output_ptr`, returns `actual_output_size`
6. Host reads result from `output_ptr`
7. Guest calls `deallocate()` for both buffers when safe

### Receivers

#### Interface
```wasm
;; Start receiver with configuration
(func (export "traces_receiver.start") (result i32))

;; Stop receiver gracefully
(func (export "traces_receiver.stop") (result i32))

;; Check if receiver is active
(func (export "traces_receiver.is_active") (result i32))
```

#### Host Functions for Receivers
```wasm
;; Send received data to pipeline
(func (import "otelwasm.v2.data" "send_traces")
  (param $traces_ptr i32) (param $traces_size i32) (result i32))

;; Check if shutdown requested
(func (import "otelwasm.v2.control" "is_shutdown_requested") (result i32))
```

### Exporters

#### Interface
```wasm
;; Export data to external system
(func (export "traces_exporter.export")
  (param $data_ptr i32) (param $data_size i32) (result i32))

;; Batch export multiple datasets efficiently
(func (export "traces_exporter.export_batch")
  (param $batch_ptr i32) (param $batch_size i32) (result i32))
```

#### Data Flow
1. Host allocates data in guest memory
2. Host calls `traces_exporter.export(data_ptr, data_size)`
3. Guest reads data and exports to external system
4. Guest returns status code
5. Host reads detailed status if needed via `get_last_status()`

## Migration Strategy

### Phase 1: Backward Compatibility Layer
- Implement ABI v2 alongside v1
- Create adapter layer that translates v1 calls to v2
- Existing plugins continue to work unchanged

### Phase 2: New Plugin Development
- New plugins use ABI v2 exclusively
- Provide rich SDKs for Go, Rust, C++, JavaScript/AssemblyScript
- Extensive documentation and examples

### Phase 3: Legacy Migration
- Provide migration tools for v1 → v2
- Deprecation timeline for v1 features
- Performance incentives for v2 adoption

### Phase 4: V1 Sunset
- Remove v1 compatibility layer
- Pure v2 implementation for optimal performance

## Implementation Considerations

### Host Implementation Changes
1. **Memory Management**: Update wazero integration for guest-managed allocation
2. **Status Handling**: Implement structured status retrieval
3. **Streaming Support**: Add chunked data transfer mechanisms
4. **Service Discovery**: Implement capability negotiation
5. **Configuration**: Dynamic config updates via lifecycle hooks

### Guest SDK Changes
1. **Memory Allocators**: Provide efficient allocators for different languages
2. **Streaming Helpers**: High-level APIs for chunked processing
3. **Status Builders**: Convenience functions for structured error reporting
4. **Service Templates**: Boilerplate generators for different component types

### Performance Implications
- **Memory**: Reduced copying with guest-managed allocation
- **CPU**: Streaming reduces peak memory usage and processing time
- **Network**: Lazy loading reduces unnecessary data transfer
- **Debugging**: Structured errors improve troubleshooting time

## Security Considerations

1. **Memory Safety**: Guest-managed allocation prevents buffer overflows
2. **Resource Limits**: Streaming prevents excessive memory consumption
3. **Sandboxing**: Maintain WASM sandbox integrity
4. **Input Validation**: Structured data validation at ABI boundaries

## Timeline Estimate

- **Design & Prototyping**: 2-3 months
- **Core Implementation**: 3-4 months
- **SDK & Tooling**: 2-3 months
- **Testing & Documentation**: 1-2 months
- **Migration Support**: 2-3 months

**Total**: 10-15 months for full ABI v2 implementation

## Conclusion

ABI v2 addresses all critical limitations of the current system while providing a robust foundation for future growth. The design leverages proven patterns from proxy-wasm and kube-scheduler-wasm-extension while staying true to OpenTelemetry's specific needs.

Key benefits:
- **Safety**: Guest-managed memory eliminates buffer overflow risks
- **Performance**: Streaming support enables high-volume telemetry processing
- **Extensibility**: Service-oriented design allows easy addition of new features
- **Developer Experience**: Rich error information and auto-generated bindings
- **Future-Proof**: Formal versioning and capability negotiation

The migration strategy ensures backward compatibility during the transition while providing strong incentives to adopt the improved v2 interface.