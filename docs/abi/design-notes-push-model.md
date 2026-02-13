# Design Notes: Push Model for ABI v1

**Date**: 2025-02 (initial discussion)
**Status**: Design decision recorded
**Related**: [ABI v1 Specification](v1.md), [Issue #17](https://github.com/otelwasm/otelwasm/issues/17)

This document captures the design rationale and implementation strategy for adopting
the **push model** in the otelwasm ABI v1. It is intended as a reference for future
contributors implementing the v1 ABI in the host runtime and guest SDKs.

---

## 1. Background: Pull Model (Experimental ABI)

The experimental ABI uses a **pull model** where the guest actively requests telemetry
data from the host during processing:

```
Host                                  Guest
 │                                     │
 │── processTraces() ─────────────────►│
 │                                     │── currentTraces(buf, limit) ──►│
 │◄── marshal + memory.Write ──────────│                                │
 │                                     │◄── actual_size ────────────────│
 │                                     │
 │                                     │  (process data)
 │                                     │
 │                                     │── setResultTraces(ptr, size) ──►│
 │◄── memory.Read + unmarshal ─────────│                                │
 │◄── return statusCode ──────────────│
```

Key characteristics:
- Guest pre-allocates a reusable buffer (`readBuf`, default 2048 bytes)
- Two-pass retry: if data exceeds buffer, host returns actual size, guest grows buffer and retries
- Guest controls its own memory lifecycle — no GC concerns
- Host functions `currentTraces`/`currentMetrics`/`currentLogs` serve as data accessors

See `guest/internal/mem/mem.go` for the buffer management implementation.

## 2. Decision: Adopt Push Model

We decided to adopt a **push model** for ABI v1. The host serializes telemetry data,
allocates memory in the guest via `alloc()`, writes the data, and calls the guest's
consumer function with the pointer and size.

### 2.1 Primary Motivation: Multi-Language SDK Support

The project plans to provide guest SDKs for **Go**, **Rust**, and **Zig**. The push
model is the standard WASM ABI pattern for host-to-guest data transfer and is
significantly more natural for non-GC languages:

| Language | Pull Model (`currentTraces` + two-pass buffer) | Push Model (`alloc` + `consume_traces`) |
|----------|------------------------------------------------|----------------------------------------|
| **Go**   | Natural (current implementation)               | Requires GC pinning (see §3)           |
| **Rust** | Verbose (manual buffer + retry logic)          | Natural (`std::alloc::alloc`)          |
| **Zig**  | Verbose (manual buffer + retry logic)          | Natural (`allocator.alloc`)            |

For Rust and Zig, the pull model requires implementing the two-pass buffer retry
protocol, which is non-trivial boilerplate. The push model only requires exporting a
standard `alloc` function, which is idiomatic in both languages.

### 2.2 Alignment with OpenTelemetry Collector Interfaces

The push model directly maps to the Collector's consumer interfaces:

```
consumer.ConsumeTraces(ctx, ptrace.Traces)  →  consume_traces(data_ptr, data_size)
```

This also enables a **unified interface** for processors and exporters — both implement
`consume_*()`. In the pull model, processors used `processTraces()` and exporters used
`pushTraces()`, requiring separate function names.

### 2.3 Other Considerations

| Aspect | Pull Model | Push Model |
|--------|-----------|-----------|
| Serialization timing | Lazy (on demand) | Eager (before call) |
| Host→Guest calls per invocation | 1 (consumer fn) | 2 (alloc + consumer fn) |
| Guest→Host calls per invocation | 2-3 (get data + set result) | 1 (set result) |
| Re-entrancy requirement | Yes (host→guest→host) | No |
| Standard WASM pattern | Uncommon | Common (proxy-wasm, etc.) |

The pull model has the advantage of lazy serialization — if the guest decides not to
process data (e.g., based on config), the serialization cost is avoided. However, in
practice, processors and exporters almost always need the full telemetry data, making
this advantage marginal.

## 3. Go Guest SDK: GC-Safe `alloc` Implementation

The primary implementation challenge for the push model in Go is preventing the garbage
collector from reclaiming memory allocated by `alloc()` before the host writes to it.

### 3.1 The Problem

```go
// UNSAFE: buf has no live reference after alloc returns
//go:wasmexport alloc
func alloc(size uint32) uint32 {
    buf := make([]byte, size)       // allocated on Go heap
    return uint32(uintptr(unsafe.Pointer(&buf[0])))
    // buf goes out of scope → GC may collect before host writes
}
```

### 3.2 Why It "Mostly Works" Without Pinning

WASM is single-threaded, and Go's GC only triggers during allocation (`make`, `new`,
etc.). The host-side sequence between `alloc()` and `consume_*()` is:

```
1. Host calls alloc(size)       → Guest executes, returns ptr
2. Host calls memory.Write(…)   → Direct memory write, no guest code runs
3. Host calls consume_*(ptr, …) → Guest executes, reads data
```

Between steps 1 and 3, **no guest code executes**, so GC cannot run. This is why
the simple pattern works in proxy-wasm-go-sdk for most workloads. However, this is
an implementation detail of the current runtime, not a guaranteed contract.

### 3.3 Recommended Pattern: Pinning Map

For correctness, we adopt the **pinning map** pattern referenced in TinyGo community
discussions (tinygo-org/tinygo#2187):

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
    pinnedAllocations[ptr] = buf  // prevent GC collection
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
    delete(pinnedAllocations, key) // allow GC to collect after use
    return buf[:size]
}
```

This pattern:
- Guarantees GC safety regardless of runtime behavior
- Has minimal overhead (one map insert/delete per telemetry batch)
- Is safe for concurrent use (WASM is single-threaded)

### 3.4 Reference: proxy-wasm-go-sdk

The [proxy-wasm-go-sdk](https://github.com/proxy-wasm/proxy-wasm-go-sdk) uses the
simple pattern (no pinning) in `proxywasm/internal/abi_callback_alloc.go`:

```go
//go:wasmexport proxy_on_memory_allocate
func proxyOnMemoryAllocate(size uint32) *byte {
    buf := make([]byte, size)
    return &buf[0]
}
```

Known issues with this approach are documented in:
- tetratelabs/proxy-wasm-go-sdk#5 (memory ownership)
- tetratelabs/proxy-wasm-go-sdk#349 (memory leaks)
- tinygo-org/tinygo#2187 (GC and exported alloc)

Alternative approaches considered but not adopted for otelwasm:
- **`gc=leaking`**: No GC at all. Simple but causes unbounded memory growth. TinyGo-only.
- **nottinygc (bdwgc)**: Replaces TinyGo GC with Boehm GC. TinyGo-only; otelwasm uses standard Go.
- **C.malloc via cgo**: Allocates outside Go heap. Not available in WASM target.

## 4. Rust / Zig Guest SDK: `alloc` Implementation

For non-GC languages, `alloc` is straightforward.

### 4.1 Rust

```rust
use std::alloc::{alloc, Layout};

#[no_mangle]
pub extern "C" fn alloc(size: i32) -> i32 {
    let Ok(layout) = Layout::from_size_align(size as usize, 1) else {
        return 0; // allocation failure
    };
    let ptr = unsafe { alloc(layout) };
    if ptr.is_null() {
        return 0;
    }
    ptr as i32
}
```

Memory is freed by the guest after deserialization. The guest SDK would wrap this in a
safe API that handles deallocation automatically.

### 4.2 Zig

```zig
const std = @import("std");

var gpa = std.heap.wasm_allocator;

export fn alloc(size: i32) i32 {
    const slice = gpa.alloc(u8, @intCast(size)) catch return 0;
    return @intCast(@intFromPtr(slice.ptr));
}
```

## 5. Host-Side Changes

### 5.1 New Host Flow (Push Model)

```go
func (wp *wasmProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
    // 1. Serialize telemetry data
    marshaler := ptrace.ProtoMarshaler{}
    data, err := marshaler.MarshalTraces(td)
    if err != nil {
        return td, fmt.Errorf("marshal traces: %w", err)
    }

    // 2. Allocate memory in guest
    results, err := wp.plugin.CallFunction(ctx, "alloc", uint64(len(data)))
    if err != nil {
        return td, fmt.Errorf("alloc: %w", err)
    }
    ptr := uint32(results[0])
    if ptr == 0 {
        return td, fmt.Errorf("alloc returned null for %d bytes", len(data))
    }

    // 3. Write data to guest memory
    if !wp.plugin.Memory().Write(ptr, data) {
        return td, fmt.Errorf("memory write failed at %d (%d bytes)", ptr, len(data))
    }

    // 4. Call consumer function with pointer and size
    stack := &wasmplugin.Stack{PluginConfigJSON: wp.plugin.PluginConfigJSON}
    results, err = wp.plugin.ProcessFunctionCall(ctx, "consume_traces", stack,
        uint64(ptr), uint64(len(data)))
    if err != nil {
        return td, err
    }

    // 5. Check status and return results
    statusCode := wasmplugin.StatusCode(results[0])
    if statusCode != 0 {
        return td, fmt.Errorf("error: %s: %s", statusCode.String(), stack.StatusReason)
    }
    return stack.ResultTraces, nil
}
```

### 5.2 Host Functions Removed

The following pull-model host functions are removed in v1:
- `currentTraces(buf, limit) -> actual_size`
- `currentMetrics(buf, limit) -> actual_size`
- `currentLogs(buf, limit) -> actual_size`

### 5.3 Host Functions Retained (with snake_case rename)

- `set_result_traces(ptr, size)` — unchanged semantics
- `set_result_metrics(ptr, size)` — unchanged semantics
- `set_result_logs(ptr, size)` — unchanged semantics
- `get_plugin_config(buf, limit) -> actual_size` — still uses pull/buffer-passing (§4.3 of spec)
- `set_status_reason(ptr, size)` — unchanged semantics
- `get_shutdown_requested() -> i32` — unchanged semantics

### 5.4 Host Functions Added

- `log(level, msg_ptr, msg_size)` — new structured logging

## 6. Guest-Side Changes (Go SDK)

### 6.1 Current: Pull Model

```go
// tracesprocessor.go (experimental)
//go:wasmexport processTraces
func _processTraces() uint32 {
    traces := imports.CurrentTraces()           // pull from host
    result, status := tracesprocessor.ProcessTraces(traces)
    if result != (ptrace.Traces{}) {
        pubimports.SetResultTraces(result)
    }
    runtime.KeepAlive(result)
    return imports.StatusToCode(status)
}
```

### 6.2 New: Push Model

```go
// tracesprocessor.go (v1)
//go:wasmexport consume_traces
func _consumeTraces(dataPtr uint32, dataSize uint32) uint32 {
    raw := mem.TakeOwnership(dataPtr, dataSize) // unpin + get buffer
    unmarshaler := ptrace.ProtoUnmarshaler{}
    traces, err := unmarshaler.UnmarshalTraces(raw)
    if err != nil {
        return uint32(StatusError)
    }
    result, status := tracesprocessor.ProcessTraces(traces)
    if result != (ptrace.Traces{}) {
        pubimports.SetResultTraces(result)
    }
    runtime.KeepAlive(result)
    return imports.StatusToCode(status)
}
```

### 6.3 Plugin Author API: No Change

The plugin author's interface remains unchanged regardless of the underlying ABI model:

```go
type TracesProcessor interface {
    ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *api.Status)
}
```

The guest SDK absorbs the ABI difference, so plugin authors do not need to modify their
code when migrating from experimental ABI to v1.

## 7. Migration Path

### 7.1 Dual ABI Support in Host

During the transition period, the host supports both ABIs:

```go
func detectABI(mod api.Module) ABIVersion {
    if _, ok := mod.ExportedFunction("abi_version_v1"); ok {
        return ABIV1  // push model
    }
    // Fall back to experimental ABI (pull model)
    return ABIExperimental
}
```

### 7.2 Implementation Order

Suggested implementation sequence:

1. **Guest `alloc` + pinning map** (`guest/internal/mem/alloc.go`)
2. **Guest consumer exports** (`consume_traces/metrics/logs` with `TakeOwnership`)
3. **Host push flow** (serialize → alloc → write → call consumer)
4. **Remove pull host functions** (`currentTraces/Metrics/Logs`)
5. **Host ABI detection** (dual support for experimental + v1)
6. **snake_case rename** for all functions
7. **Guest SDK for Rust** (separate repository)
8. **Guest SDK for Zig** (separate repository)

## 8. Open Questions

- **`alloc` failure handling**: Should the host retry with a smaller batch, or
  immediately propagate the error? The current spec says propagate, but retry may be
  desirable for large batches.
- **`dealloc` export**: Should the guest also export `dealloc(ptr, size)` for
  symmetry? Rust's allocator requires knowing the layout for deallocation. For v1,
  deallocation is handled internally by the guest after `consume_*` returns, so
  `dealloc` is not needed by the host. This may change if future features require the
  host to manage guest memory lifetime.
- **Receiver model**: Receivers use `start_*_receiver()` which blocks and does not
  receive pushed data. This is orthogonal to the push/pull decision for processors and
  exporters. Future versions may introduce a tick-based receiver model.
