# ABI v1 Migration Design

**Date**: 2026-02-15
**Status**: Approved
**Related**: [ABI v1 Specification](../abi/v1.md), [Push Model Design Notes](../abi/design-notes-push-model.md)

---

## 1. Overview

This document describes the design for migrating otelwasm from the experimental ABI to ABI v1. The migration adopts the **push model** for host-to-guest data transfer, aligns with OpenTelemetry Collector interfaces, and implements the complete ABI v1 specification.

**Scope**: Complete migration to ABI v1 with removal of experimental ABI support.

**Components**: All component types (Processor, Exporter, Receiver) across all signals (traces, metrics, logs).

**Implementation**: Phased approach with Phase 1 focusing on push model + lifecycle functions.

---

## 2. Background and Motivation

### 2.1 Current State (Experimental ABI)

The experimental ABI uses a **pull model**:
- Guest calls `currentTraces(buf, limit)` to pull data from host
- Two-pass buffer retry pattern for variable-size data
- Function names use camelCase (`processTraces`, `setResultTraces`)
- No explicit lifecycle management (`start`/`shutdown`)
- Module name: `opentelemetry.io/wasm`

### 2.2 Why ABI v1?

1. **Multi-language SDK support**: Push model is standard in WASM ABIs and more natural for Rust/Zig
2. **OTel Collector alignment**: Direct mapping to `consumer.ConsumeTraces` and `component.Component` interfaces
3. **Unified interface**: Processors and exporters both implement `consume_*()` (vs separate `process*`/`push*` in experimental)
4. **Lifecycle management**: Explicit `start()`/`shutdown()` phases matching Collector component lifecycle
5. **Industry standards**: Follows patterns from proxy-wasm and http-wasm

### 2.3 Migration Goals

- ✅ Complete ABI v1 implementation
- ✅ Remove experimental ABI support (clean refactor)
- ✅ Maintain existing plugin author API (no breaking changes for user code)
- ✅ Support all component types and signal types (9 patterns)
- ✅ Comprehensive testing and examples

---

## 3. Requirements

### 3.1 Functional Requirements

1. **ABI v1 Compliance**:
   - Push model with `alloc()` and `consume_*(ptr, size)`
   - ABI version detection via `abi_version_v1` marker
   - snake_case function naming
   - Lifecycle functions: `start()`, `shutdown()`
   - Status codes and error reporting via `set_status_reason`

2. **Component Support**:
   - **Processor**: `consume_traces/metrics/logs`
   - **Exporter**: `consume_traces/metrics/logs`
   - **Receiver**: `start_traces/metrics/logs_receiver` (blocking model)

3. **Memory Management**:
   - Go guest SDK: Pinning map to prevent GC collection
   - Host allocates via guest's `alloc()` before pushing data
   - Guest unpins and reclaims memory via `TakeOwnership()`

4. **Module Name**:
   - **Decision**: Keep `opentelemetry.io/wasm` (deviation from spec's `otelwasm`)
   - **Rationale**: Maintain consistency and avoid unnecessary churn

### 3.2 Non-Functional Requirements

1. **Performance**: Minimal overhead from push model (2 calls: alloc + consume vs 1 in pull)
2. **Testability**: All 9 component×signal patterns tested
3. **Maintainability**: Clean code structure, clear separation of concerns
4. **Compatibility**: Plugin author API unchanged (internal ABI changes only)

### 3.3 Phase 1 Scope (This Implementation)

**Included**:
- ✅ Push model (alloc → write → consume)
- ✅ Lifecycle functions (`start`, `shutdown`)
- ✅ Pinning map for Go GC safety
- ✅ snake_case function renaming
- ✅ ABI v1 marker detection

**Deferred to Phase 2**:
- ⏳ `log()` host function (structured logging)

---

## 4. Architecture Design

### 4.1 System Layers

```
┌─────────────────────────────────────────────────────┐
│  OTel Collector Pipeline                            │
│  (traces/metrics/logs consumer interfaces)         │
└─────────────────┬───────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────┐
│  Components Layer (wasmprocessor/exporter/receiver) │
│  - Call consume_traces/metrics/logs()               │
│  - Manage start()/shutdown() lifecycle              │
└─────────────────┬───────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────┐
│  Host Runtime Layer (wasmplugin/)                   │
│  - ABI detection (abi_version_v1 marker)            │
│  - Push model: alloc() → write → consume_*()        │
│  - Host functions: set_result_*, get_plugin_config  │
│  - Module name: "opentelemetry.io/wasm"             │
└─────────────────┬───────────────────────────────────┘
                  │ WASM boundary
┌─────────────────▼───────────────────────────────────┐
│  Guest SDK Layer (guest/)                           │
│  - Pinning map (mem.Alloc/TakeOwnership)            │
│  - Export: consume_*, start_*_receiver              │
│  - Wrap user plugin implementations                 │
└─────────────────┬───────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────┐
│  Plugin Implementation                              │
│  (User code, API unchanged)                         │
└─────────────────────────────────────────────────────┘
```

### 4.2 Migration Approach: Hybrid Strategy

**Strategy**: Mix of direct replacement and new implementation based on component characteristics.

1. **wasmplugin/** (Host Runtime):
   - **Approach**: Direct replacement of existing files
   - **Rationale**: Core ABI logic benefits from clean rewrite
   - **Files**: `plugin.go`, `memory.go` (edit), `abi.go` (new)

2. **guest/** (Guest SDK):
   - **Approach**: New implementation with gradual migration
   - **Rationale**: Memory management fundamentally changes (pull → push + pinning map)
   - **Files**:
     - `guest/internal/mem/alloc.go` (new) - Pinning map
     - `guest/internal/mem/mem.go` (keep) - Buffer passing for `get_plugin_config`
     - `guest/plugin/extern.go` (edit) - snake_case exports
     - Component packages (edit) - consume_* exports

3. **components/** (Processor/Exporter/Receiver):
   - **Approach**: Direct replacement
   - **Rationale**: Interface changes only (function names + lifecycle calls)
   - **Files**: Existing test files updated

---

## 5. Major Components Design

### 5.1 Host Runtime Layer (wasmplugin/)

#### 5.1.1 ABI Detection

**New file**: `wasmplugin/abi.go`

```go
type ABIVersion int

const (
    ABIUnknown ABIVersion = iota
    ABIV1
)

func detectABIVersion(mod api.Module) ABIVersion {
    if mod.ExportedFunction("abi_version_v1") != nil {
        return ABIV1
    }
    return ABIUnknown
}
```

#### 5.1.2 Push Model Implementation

**Modified**: `wasmplugin/plugin.go`

```go
func (p *WasmPlugin) ConsumeTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
    // 1. Serialize telemetry data
    marshaler := ptrace.ProtoMarshaler{}
    data, err := marshaler.MarshalTraces(td)
    if err != nil {
        return td, fmt.Errorf("marshal traces: %w", err)
    }

    // 2. Allocate memory in guest
    results, err := p.Module.ExportedFunction("alloc").Call(ctx, uint64(len(data)))
    if err != nil {
        return td, fmt.Errorf("alloc: %w", err)
    }
    ptr := uint32(results[0])
    if ptr == 0 {
        return td, fmt.Errorf("alloc returned null for %d bytes", len(data))
    }

    // 3. Write data to guest memory
    if !p.Module.Memory().Write(ptr, data) {
        return td, fmt.Errorf("memory write failed at %d (%d bytes)", ptr, len(data))
    }

    // 4. Call consumer function
    stack := &Stack{PluginConfigJSON: p.PluginConfigJSON}
    results, err = p.ProcessFunctionCall(ctx, "consume_traces", stack, uint64(ptr), uint64(len(data)))
    if err != nil {
        return td, err
    }

    // 5. Check status and return results
    statusCode := StatusCode(results[0])
    if statusCode != 0 {
        return td, fmt.Errorf("error: %s: %s", statusCode.String(), stack.StatusReason)
    }
    return stack.ResultTraces, nil
}
```

#### 5.1.3 Host Functions (snake_case)

**Modified**: `wasmplugin/plugin.go` - `instantiateHostModule()`

**Removed**:
- `currentTraces(buf, limit) -> size`
- `currentMetrics(buf, limit) -> size`
- `currentLogs(buf, limit) -> size`

**Renamed** (camelCase → snake_case):
- `setResultTraces` → `set_result_traces`
- `setResultMetrics` → `set_result_metrics`
- `setResultLogs` → `set_result_logs`
- `getPluginConfig` → `get_plugin_config`
- `setResultStatusReason` → `set_status_reason`
- `getShutdownRequested` → `get_shutdown_requested`

**Module name**: `opentelemetry.io/wasm` (unchanged)

#### 5.1.4 Lifecycle Support

**New exports required**:
- `start() -> i32` - Called once during component startup
- `shutdown() -> i32` - Called once during component shutdown

Host calls these functions at appropriate lifecycle phases.

### 5.2 Guest SDK Layer (guest/)

#### 5.2.1 Pinning Map

**New file**: `guest/internal/mem/alloc.go`

```go
package mem

import "unsafe"

// pinnedAllocations keeps references to host-allocated buffers,
// preventing Go GC from collecting them.
var pinnedAllocations = make(map[uintptr][]byte)

//go:wasmexport alloc
func Alloc(size uint32) uint32 {
    buf := make([]byte, size)
    ptr := uintptr(unsafe.Pointer(&buf[0]))
    pinnedAllocations[ptr] = buf  // Prevent GC collection
    return uint32(ptr)
}

// TakeOwnership retrieves and unpins a buffer allocated by alloc().
// Called by consume_* functions after reading the data.
func TakeOwnership(ptr uint32, size uint32) []byte {
    key := uintptr(ptr)
    buf, ok := pinnedAllocations[key]
    if !ok {
        panic("TakeOwnership: unknown pointer")
    }
    delete(pinnedAllocations, key) // Allow GC to collect after use
    return buf[:size]
}
```

**Rationale**: See [design-notes-push-model.md](../abi/design-notes-push-model.md) §3 for detailed explanation of why pinning is necessary for Go's GC.

#### 5.2.2 Consumer Exports

**Modified**: `guest/tracesprocessor/tracesprocessor.go`

```go
//go:wasmexport consume_traces
func _consumeTraces(dataPtr uint32, dataSize uint32) uint32 {
    // Unpin + get buffer
    raw := mem.TakeOwnership(dataPtr, dataSize)

    // Deserialize
    unmarshaler := ptrace.ProtoUnmarshaler{}
    traces, err := unmarshaler.UnmarshalTraces(raw)
    if err != nil {
        imports.SetStatusReason(err.Error())
        return uint32(api.StatusError)
    }

    // Call user implementation
    result, status := tracesprocessor.ProcessTraces(traces)

    // Set result
    if result != (ptrace.Traces{}) {
        imports.SetResultTraces(result)
    }

    runtime.KeepAlive(result)
    return imports.StatusToCode(status)
}
```

Similar implementation for:
- `consume_metrics` (metricsprocessor/metricsexporter)
- `consume_logs` (logsprocessor/logsexporter)

#### 5.2.3 Receiver Exports

**Modified**: `guest/tracesreceiver/tracesreceiver.go`

```go
//go:wasmexport start_traces_receiver
func _startTracesReceiver() {
    for {
        // Generate/collect telemetry
        traces := tracesreceiver.GenerateTraces()

        // Emit to pipeline
        imports.SetResultTraces(traces)

        // Check shutdown
        if imports.GetShutdownRequested() {
            break
        }

        // Sleep/wait
        time.Sleep(1 * time.Second)
    }
}
```

#### 5.2.4 Lifecycle Exports

**Modified**: `guest/plugin/extern.go`

```go
//go:wasmexport abi_version_v1
func _abiVersionV1() {
    // Empty function - marker only
}

//go:wasmexport start
func _start() uint32 {
    // Plugin initialization
    // Can call get_plugin_config() here
    return 0 // SUCCESS
}

//go:wasmexport shutdown
func _shutdown() uint32 {
    // Cleanup resources
    return 0 // SUCCESS
}
```

#### 5.2.5 Import Renaming

**Modified**: `guest/imports/host.go`

Rename all import declarations to snake_case:

```go
//go:wasmimport opentelemetry.io/wasm set_result_traces
func setResultTraces(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm set_result_metrics
func setResultMetrics(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm set_result_logs
func setResultLogs(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm get_plugin_config
func getPluginConfig(ptr, limit uint32) uint32

//go:wasmimport opentelemetry.io/wasm set_status_reason
func setStatusReason(ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm get_shutdown_requested
func getShutdownRequested() uint32
```

### 5.3 Components Layer

#### 5.3.1 Processor

**Modified**: `wasmprocessor/processor.go`

```go
const (
    consumeTracesFunctionName  = "consume_traces"
    consumeMetricsFunctionName = "consume_metrics"
    consumeLogsFunctionName    = "consume_logs"
    startFunctionName          = "start"
    shutdownFunctionName       = "shutdown"
)

func newWasmTracesProcessor(ctx context.Context, cfg *Config) (*wasmProcessor, error) {
    // ... validation ...

    requiredFunctions := []string{
        consumeTracesFunctionName,
        startFunctionName,
        shutdownFunctionName,
    }

    plugin, err := wasmplugin.NewWasmPlugin(ctx, &cfg.Config, requiredFunctions)
    if err != nil {
        return nil, err
    }

    // Call start() lifecycle function
    if err := plugin.Start(ctx); err != nil {
        return nil, fmt.Errorf("failed to start plugin: %w", err)
    }

    return &wasmProcessor{plugin: plugin}, nil
}

func (wp *wasmProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
    // Use push model from wasmplugin
    return wp.plugin.ConsumeTraces(ctx, td)
}

func (wp *wasmProcessor) shutdown(ctx context.Context) error {
    // Call shutdown() lifecycle function
    if err := wp.plugin.Shutdown(ctx); err != nil {
        return err
    }
    return wp.plugin.Close(ctx)
}
```

#### 5.3.2 Exporter

**Modified**: `wasmexporter/exporter.go`

Similar to processor, but with exporter-specific logic:
- Same function names (`consume_traces`, etc.)
- No `set_result_*` expected (terminal stage)
- Status code indicates export success/failure

#### 5.3.3 Receiver

**Modified**: `wasmreceiver/receiver.go`

```go
const (
    startTracesReceiverFunctionName  = "start_traces_receiver"
    startMetricsReceiverFunctionName = "start_metrics_receiver"
    startLogsReceiverFunctionName    = "start_logs_receiver"
    startFunctionName                = "start"
    shutdownFunctionName             = "shutdown"
)

func (wr *wasmReceiver) Start(ctx context.Context, host component.Host) error {
    // Call start() lifecycle function
    if err := wr.plugin.Start(ctx); err != nil {
        return err
    }

    // Start receiver in goroutine (blocking call)
    go func() {
        stack := &wasmplugin.Stack{
            PluginConfigJSON: wr.plugin.PluginConfigJSON,
            OnResultTracesChange: func(td ptrace.Traces) {
                _ = wr.nextConsumer.ConsumeTraces(ctx, td)
            },
        }
        _, _ = wr.plugin.ProcessFunctionCall(ctx, startTracesReceiverFunctionName, stack)
    }()

    return nil
}

func (wr *wasmReceiver) Shutdown(ctx context.Context) error {
    // Set shutdown flag
    wr.plugin.RequestShutdown()

    // Call shutdown() lifecycle function
    if err := wr.plugin.Shutdown(ctx); err != nil {
        return err
    }

    return wr.plugin.Close(ctx)
}
```

---

## 6. Data Flow

### 6.1 Processor Data Flow

```
OTel Collector Pipeline
    ↓ ConsumeTraces(ctx, ptrace.Traces)
    ↓
wasmprocessor
    ↓ 1. Marshal traces to protobuf
    ↓ 2. Call guest.alloc(size) → ptr
    ↓ 3. Memory.Write(ptr, data)
    ↓ 4. Call guest.consume_traces(ptr, size) → statusCode
    ↓
WASM Guest
    ↓ consume_traces(ptr, size):
    ↓   - TakeOwnership(ptr, size) → unpin & get buffer
    ↓   - Unmarshal protobuf → ptrace.Traces
    ↓   - Call user's ProcessTraces()
    ↓   - Marshal result → call set_result_traces(ptr, size)
    ↓   - Return statusCode
    ↓
wasmprocessor
    ↓ 5. Read stack.ResultTraces
    ↓ 6. Return to pipeline
    ↓
OTel Collector Pipeline
```

### 6.2 Receiver Data Flow (Blocking Model)

```
wasmreceiver.Start()
    ↓ Call guest.start()
    ↓ Start goroutine
    ↓   Call guest.start_traces_receiver()
    ↓
WASM Guest (blocking loop)
    ↓ loop:
    ↓   - Generate/collect telemetry data
    ↓   - Marshal → call set_result_traces(ptr, size)
    ↓   - Check get_shutdown_requested()
    ↓   - if shutdown: break
    ↓
wasmreceiver
    ↓ OnResultTracesChange callback
    ↓   → nextConsumer.ConsumeTraces(ctx, traces)
    ↓
OTel Collector Pipeline
```

### 6.3 Lifecycle Flow

**Startup:**
```
Component.Start(ctx, host)
    ↓ 1. Load WASM module
    ↓ 2. Instantiate (calls _initialize if present)
    ↓ 3. Check abi_version_v1 export
    ↓ 4. Call guest.start() → statusCode
    ↓ 5. If statusCode != 0: return error
    ↓ 6. Component ready
```

**Shutdown:**
```
Component.Shutdown(ctx)
    ↓ 1. For receiver: set shutdown flag
    ↓ 2. Wait for in-flight operations
    ↓ 3. Call guest.shutdown() → statusCode
    ↓ 4. Close WASM runtime
    ↓ 5. Return error if statusCode != 0
```

---

## 7. Error Handling

### 7.1 Status Codes

ABI v1 defines:
- `SUCCESS = 0`: Operation completed successfully
- `ERROR = 1`: Operation failed (reason via `set_status_reason`)

### 7.2 Error Flow

**Guest function errors:**
```go
// Guest side
if err != nil {
    imports.SetStatusReason(err.Error())
    return uint32(api.StatusError)
}
```

```go
// Host side
statusCode := results[0]
if statusCode != 0 {
    return fmt.Errorf("guest error: %s: %s",
        StatusCode(statusCode), stack.StatusReason)
}
```

**Allocation failures:**
```go
ptr := p.callAlloc(size)
if ptr == 0 {
    return nil, fmt.Errorf("guest alloc failed for %d bytes", size)
}
```

**Memory access errors:**
```go
if !p.Module.Memory().Write(ptr, data) {
    return nil, fmt.Errorf("memory write failed at %d", ptr)
}
```

**WASM traps:**
- Wazero runtime automatically returns errors
- Host logs error and propagates to pipeline

---

## 8. Test Strategy

### 8.1 Test Coverage

**Layer-based testing:**

1. **wasmplugin layer**:
   - ABI detection unit tests
   - Push model (alloc → write → consume) tests
   - Error handling (alloc failure, memory errors)
   - Extend existing `wasmplugin/config_test.go`

2. **guest SDK layer**:
   - Pinning map tests (GC safety)
   - `Alloc`/`TakeOwnership` memory management tests
   - Extend existing `guest/internal/plugin/plugin_test.go`

3. **components layer**:
   - Integration tests for all 9 patterns (component × signal)
   - Update existing test files:
     - `wasmprocessor/processor_test.go`
     - `wasmexporter/exporter_test.go`
     - `wasmreceiver/receiver_test.go`

### 8.2 WASM Example Migration

**All 9 patterns migrated to ABI v1:**

| Component | Traces | Metrics | Logs |
|-----------|--------|---------|------|
| Processor | ✓ | ✓ | ✓ |
| Exporter  | ✓ | ✓ | ✓ |
| Receiver  | ✓ | ✓ | ✓ |

**Example implementations:**
- **Processor**: Add/modify attributes (maintains existing behavior)
- **Exporter**: stdout output (maintains existing behavior)
- **Receiver**: Sample data generation (maintains existing behavior)

**Required changes:**
- Add `abi_version_v1()` marker
- Add `alloc()` export
- Rename functions to `consume_*()` / `start_*_receiver()`
- Add `start()`/`shutdown()` lifecycle

### 8.3 Test Execution

```bash
# 1. Build WASM examples
make build-wasm-examples

# 2. Copy to testdata
make copy-wasm-examples

# 3. Run tests (requires Docker tag)
make test
# Or per-module:
cd wasmprocessor && go test -tags docker -v ./...
```

**Test phases:**
1. **Phase 1** (Foundation): wasmplugin + guest SDK unit tests
2. **Phase 2** (Processor): 3 signal integration tests
3. **Phase 3** (Exporter): 3 signal integration tests
4. **Phase 4** (Receiver): 3 signal integration tests

### 8.4 Performance Benchmarking

**Critical for validating non-functional requirements:**

Add benchmarks to existing `benchmark_test.go` files:
- Measure push model overhead (alloc → write → consume)
- Compare against experimental ABI baseline
- Monitor memory allocation patterns
- Track throughput (batches/second)

**Target**: Push model overhead < 5% compared to experimental ABI

### 8.5 Fuzz Testing

**ABI boundary is prime target for fuzzing:**

```go
func FuzzConsumeTraces(f *testing.F) {
    // Test guest's unmarshalling with random/malformed data
    // Ensures no panics or security issues
}
```

Use Go's native fuzzing for:
- Malformed protobuf data
- Invalid pointer/size combinations
- Edge cases in data structures

### 8.6 Negative Testing

**Explicit failure scenario tests:**

1. **Missing ABI marker**: Load WASM without `abi_version_v1` export
2. **Missing required functions**: Load module missing `start`, `consume_traces`, etc.
3. **Allocation failures**: Test when `alloc()` returns 0
4. **Status code propagation**: Verify host logs errors from `set_status_reason`
5. **Memory limit**: Test payloads larger than guest memory

### 8.7 Verification Checklist

Each test verifies:
- ✅ ABI v1 marker detected
- ✅ `alloc()` allocates memory correctly
- ✅ Data pushed correctly
- ✅ `consume_*`/`start_*_receiver` called correctly
- ✅ Results returned correctly
- ✅ Errors handled appropriately
- ✅ `start()`/`shutdown()` work correctly
- ✅ No memory leaks (pinning map works)
- ✅ Performance within acceptable range
- ✅ Fuzz testing passes without panics
- ✅ Negative tests handle errors gracefully

---

## 9. Implementation Plan

### 9.1 Implementation Order

**Vertical Slice Approach (Recommended by Gemini):**

**Phase A: Complete Traces Processor End-to-End**
1. Foundation (wasmplugin + guest SDK):
   - wasmplugin: ABI detection, push model infrastructure
   - guest SDK: pinning map (`alloc.go`)
   - Host functions: snake_case rename

2. Traces Processor (full vertical slice):
   - wasmprocessor: consume_traces, lifecycle
   - guest SDK: consume_traces export
   - Example: traces processor WASM
   - Tests: full integration test + benchmarks + fuzz + negative tests

**Validation checkpoint**: Ensure entire architecture works end-to-end before proceeding.

**Phase B: Expand to All Signals & Components**
3. Remaining Processor signals (metrics → logs):
   - wasmprocessor: consume_metrics, consume_logs
   - Examples + tests for each

4. Exporter (traces → metrics → logs):
   - wasmexporter: all signals
   - Examples + tests for each

5. Receiver (traces → metrics → logs):
   - wasmreceiver: all signals
   - Examples + tests for each

**Rationale**: The vertical slice approach proves the architecture before scaling horizontally, reducing risk of discovering fundamental issues late in the migration.

### 9.2 Commit Strategy

**Option 1: Scaffolding First (Recommended)**
1. Scaffold commit: Create all new files with TODOs
2. wasmplugin: ABI detection and infrastructure
3. wasmplugin: Push model implementation
4. wasmplugin: Host functions snake_case rename
5. guest SDK: Pinning map implementation
6. guest SDK: consume_traces export + lifecycle
7. wasmprocessor: traces processor implementation
8. Examples: traces processor WASM
9. Tests: traces processor integration + benchmarks
10. (Continue with other signals/components)

**Option 2: Incremental Without Scaffolding**
- Same order but without initial skeleton
- More organic growth

**Scaffolding benefits**: Clear skeleton, easier progress tracking, better parallelization opportunities

### 9.3 Branch Strategy

- Branch: `feature/abi-v1-migration`
- Base: `main`
- Merge: PR after all tests pass

---

## 10. Risk Mitigation Details

Based on Gemini's risk assessment, here are detailed mitigation strategies:

### 10.1 Memory Safety & Leaks (Critical Risk)

**Problem**: Pinning map is fragile—bugs could cause leaks or corruption.

**Mitigation Strategies:**

1. **Defensive Coding with `defer`**:
```go
//go:wasmexport consume_traces
func _consumeTraces(dataPtr uint32, dataSize uint32) uint32 {
    // CRITICAL: Always unpin, even if function panics
    defer func() {
        if r := recover(); r != nil {
            mem.TakeOwnership(dataPtr, dataSize) // Unpin on panic
            panic(r) // Re-panic after cleanup
        }
    }()

    raw := mem.TakeOwnership(dataPtr, dataSize)
    // ... rest of function
}
```

2. **GC Stress Testing**:
```bash
# Run tests with aggressive GC
GOGC=1 go test -tags docker -v ./...

# Add explicit GC calls in tests
runtime.GC()
runtime.GC()
```

3. **Memory Leak Detection**:
```bash
# Long-running test with pprof
go test -memprofile=mem.prof -run=TestLongRunning
go tool pprof -alloc_space mem.prof
# Check for growth in pinnedAllocations map
```

### 10.2 Concurrency (Medium Risk)

**Problem**: Concurrent calls to plugin instance could cause race conditions.

**Mitigation Strategies:**

1. **Host-Side Mutex**:
```go
type WasmPlugin struct {
    mu sync.Mutex  // Serialize all calls into WASM
    // ... other fields
}

func (p *WasmPlugin) ConsumeTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    // ... implementation
}
```

2. **Contract Documentation**:
- Document in wasmplugin package that guest plugins are not required to be thread-safe
- Host is responsible for serialization

### 10.3 Performance Overhead (Low-Medium Risk)

**Problem**: Push model adds marshal → alloc → write → consume steps.

**Mitigation Strategies:**

1. **Early Benchmarking**:
```go
func BenchmarkPushModelVsExperimental(b *testing.B) {
    // Compare new vs old ABI
}

func BenchmarkPushModelVsNative(b *testing.B) {
    // Compare WASM vs native Go component
}
```

2. **Optimization Opportunities**:
- Consider buffer pooling for marshaling
- Monitor allocation patterns with pprof
- Profile hot paths with CPU profiling

### 10.4 Edge Cases

**Critical edge cases to handle:**

1. **Zero-Payload Data**:
```go
// Host side
if len(data) == 0 {
    // Skip alloc/write, just call consume with null ptr
    return p.callConsumeTraces(0, 0)
}
```

2. **Guest Memory Limits**:
```go
// Host side
ptr := p.callAlloc(size)
if ptr == 0 {
    // Retry with smaller batch or fail gracefully
    return nil, fmt.Errorf("guest out of memory (requested %d bytes)", size)
}
```

3. **Receiver Shutdown Race**:
```go
// Host side
func (wr *wasmReceiver) Shutdown(ctx context.Context) error {
    wr.plugin.RequestShutdown()

    // Close callback to drop late data
    wr.mu.Lock()
    wr.onResultTracesChange = func(td ptrace.Traces) {
        // Drop data arriving after shutdown
    }
    wr.mu.Unlock()

    // ... continue shutdown
}
```

4. **Lifecycle Idempotency**:
```go
// Host side
func (p *WasmPlugin) Start(ctx context.Context) error {
    if p.started.Swap(true) {
        return nil // Already started, no-op
    }
    // ... call guest start()
}
```

---

## 11. Deviations from ABI v1 Specification

### 11.1 Module Name

**Specification**: `otelwasm`
**Implementation**: `opentelemetry.io/wasm`

**Rationale**: Maintain consistency with existing codebase and avoid unnecessary churn. The module name is an implementation detail and does not affect ABI compatibility.

---

## 12. Phase 2 Considerations (Future Work)

**Deferred features:**

1. **`log()` host function**:
   - Structured logging with 5 log levels
   - Maps to OTel Collector logger
   - Guest SDK provides `Log(level, message)` API

**Estimated complexity**: Low (straightforward host function addition)

---

## 13. References

- [ABI v1 Specification](../abi/v1.md)
- [Push Model Design Notes](../abi/design-notes-push-model.md)
- [proxy-wasm ABI](https://github.com/proxy-wasm/spec)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [Wazero Runtime](https://wazero.io/)

---

## 14. Approval and Review

**Design approved**: 2026-02-15
**Approved by**: User
**Reviewed by**: Gemini AI (2026-02-15)

**Gemini Review Summary**:
- ✅ Design is fundamentally sound and well-researched
- ✅ Hybrid migration strategy is pragmatic
- ✅ Pinning map is the correct solution for Go GC
- ⚠️ Primary risks: memory safety, concurrency, performance
- 💡 Recommendation: Vertical slice approach (complete traces processor first)
- 💡 Recommendation: Add benchmarking, fuzz testing, negative testing

**Next step**: Proceed to implementation planning with writing-plans skill
