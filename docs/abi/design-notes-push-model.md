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
allocates memory in the guest via `otelwasm_memory_allocate()`, writes the data, and calls the guest's
consumer function with the pointer and size.

### 2.1 Primary Motivation: Multi-Language SDK Support

The project plans to provide guest SDKs for **Go**, **Rust**, and **Zig**. The push
model is the standard WASM ABI pattern for host-to-guest data transfer and is
significantly more natural for non-GC languages:

| Language | Pull Model (`currentTraces` + two-pass buffer) | Push Model (`otelwasm_memory_allocate` + `otelwasm_consume_traces`) |
|----------|------------------------------------------------|----------------------------------------|
| **Go**   | Natural (current implementation)               | Requires GC pinning (see §3)           |
| **Rust** | Verbose (manual buffer + retry logic)          | Natural (`std::alloc::alloc`)          |
| **Zig**  | Verbose (manual buffer + retry logic)          | Natural (`allocator.alloc`)            |

For Rust and Zig, the pull model requires implementing the two-pass buffer retry
protocol, which is non-trivial boilerplate. The push model only requires exporting a
standard `otelwasm_memory_allocate` function, which is idiomatic in both languages.

### 2.2 Alignment with OpenTelemetry Collector Interfaces

The push model directly maps to the Collector's consumer interfaces:

```
consumer.ConsumeTraces(ctx, ptrace.Traces)  →  otelwasm_consume_traces(data_ptr, data_size)
```

This also enables a **unified interface** for processors and exporters — both implement
`consume_*()`. In the pull model, processors used `processTraces()` and exporters used
`pushTraces()`, requiring separate function names.

### 2.3 Other Considerations

| Aspect | Pull Model | Push Model |
|--------|-----------|-----------|
| Serialization timing | Lazy (on demand) | Eager (before call) |
| Host→Guest calls per invocation | 1 (consumer fn) | 2 (otelwasm_memory_allocate + consumer fn) |
| Guest→Host calls per invocation | 2-3 (get data + set result) | 1 (set result) |
| Re-entrancy requirement | Yes (host→guest→host) | No |
| Standard WASM pattern | Uncommon | Common (proxy-wasm, etc.) |

The pull model has the advantage of lazy serialization — if the guest decides not to
process data (e.g., based on config), the serialization cost is avoided. However, in
practice, processors and exporters almost always need the full telemetry data, making
this advantage marginal.

## 3. Go Guest SDK: GC-Safe `otelwasm_memory_allocate` Implementation

The primary implementation challenge for the push model in Go is preventing the garbage
collector from reclaiming memory allocated by `otelwasm_memory_allocate()` before the host writes to it.

### 3.1 The Problem

```go
// UNSAFE: buf has no live reference after alloc returns
//go:wasmexport otelwasm_memory_allocate
func otelwasm_memory_allocate(size uint32) uint32 {
    buf := make([]byte, size)       // allocated on Go heap
    return uint32(uintptr(unsafe.Pointer(&buf[0])))
    // buf goes out of scope → GC may collect before host writes
}
```

### 3.2 Why It "Mostly Works" Without Pinning

WASM is single-threaded, and Go's GC only triggers during allocation (`make`, `new`,
etc.). The host-side sequence between `otelwasm_memory_allocate()` and `consume_*()` is:

```
1. Host calls otelwasm_memory_allocate(size)       → Guest executes, returns ptr
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

//go:wasmexport otelwasm_memory_allocate
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

## 4. Rust / Zig Guest SDK: `otelwasm_memory_allocate` Implementation

For non-GC languages, `otelwasm_memory_allocate` is straightforward.

### 4.1 Rust

```rust
use std::alloc::{alloc, Layout};

#[no_mangle]
pub extern "C" fn otelwasm_memory_allocate(size: i32) -> i32 {
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

export fn otelwasm_memory_allocate(size: i32) i32 {
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
    results, err := wp.plugin.CallFunction(ctx, "otelwasm_memory_allocate", uint64(len(data)))
    if err != nil {
        return td, fmt.Errorf("otelwasm_memory_allocate: %w", err)
    }
    ptr := uint32(results[0])
    if ptr == 0 {
        return td, fmt.Errorf("otelwasm_memory_allocate returned null for %d bytes", len(data))
    }

    // 3. Write data to guest memory
    if !wp.plugin.Memory().Write(ptr, data) {
        return td, fmt.Errorf("memory write failed at %d (%d bytes)", ptr, len(data))
    }

    // 4. Call consumer function with pointer and size
    stack := &wasmplugin.Stack{PluginConfigJSON: wp.plugin.PluginConfigJSON}
    results, err = wp.plugin.ProcessFunctionCall(ctx, "otelwasm_consume_traces", stack,
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
//go:wasmexport otelwasm_consume_traces
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
    if _, ok := mod.ExportedFunction("otelwasm_abi_version_0_1_0"); ok {
        return ABIV1  // push model
    }
    // Fall back to experimental ABI (pull model)
    return ABIExperimental
}
```

### 7.2 Implementation Order

Suggested implementation sequence:

1. **Guest `otelwasm_memory_allocate` + pinning map** (`guest/internal/mem/alloc.go`)
2. **Guest consumer exports** (`otelwasm_consume_traces/metrics/logs` with `TakeOwnership`)
3. **Host push flow** (serialize → otelwasm_memory_allocate → write → call consumer)
4. **Remove pull host functions** (`currentTraces/Metrics/Logs`)
5. **Host ABI detection** (dual support for experimental + v1)
6. **snake_case rename** for all functions
7. **Guest SDK for Rust** (separate repository)
8. **Guest SDK for Zig** (separate repository)

## 8. Open Questions

- **`otelwasm_memory_allocate` failure handling**: Should the host retry with a smaller batch, or
  immediately propagate the error? The current spec says propagate, but retry may be
  desirable for large batches.
- **`dealloc` export**: ✓ **Resolved** — After surveying other WASM ABIs (proxy-wasm,
  http-wasm, Component Model), we confirmed that omitting `dealloc` is standard practice.
  Deallocation is handled internally by the guest after `consume_*` returns. See §9 for
  detailed analysis.
- **Receiver model**: Receivers use `start_*_receiver()` which blocks and does not
  receive pushed data. This is orthogonal to the push/pull decision for processors and
  exporters. Future versions may introduce a tick-based receiver model.

---

## 9. Memory Deallocation: Industry Survey

As part of the v1 ABI design process, we surveyed how other WebAssembly ABI specifications
handle memory deallocation to validate our decision to omit an explicit `dealloc` host
function.

### 9.1 Proxy-Wasm ABI (v0.2.1 and vNEXT)

**Allocation:**
- Guest exports: `proxy_on_memory_allocate(size: i32) -> i32` (preferred)
- Guest exports: `malloc(size: i32) -> i32` (deprecated, for backward compatibility)
- Host calls these functions to allocate memory in the guest's linear memory

**Deallocation:**
- **No explicit deallocation function**
- Memory management is entirely guest-controlled
- Host requests allocations; guest manages cleanup

**Rationale (from spec):**
> "Called to allocate continuous memory buffer of `memory_size` using the in-VM memory
> allocator."

The design emphasizes plugin-controlled memory management rather than host-directed
deallocation, allowing flexible internal heap strategies.

**Source**: [proxy-wasm/spec](https://github.com/proxy-wasm/spec/blob/main/abi-versions/vNEXT/README.md)

### 9.2 HTTP-WASM Handler ABI

**Allocation:**
- **No `malloc` or `otelwasm_memory_allocate` function at all**
- Uses a buffer-passing pattern instead

**Pattern:**
- Functions accept `buf` (offset in linear memory) and `buf_limit` (max size) parameters
- Guest provides pre-allocated buffer space
- Host writes data into the specified location
- Guest can grow memory via `memory.grow` if needed

**Deallocation:**
- Not applicable (no dynamic allocation)

**Rationale (from spec):**
> "This specification relies completely on guest wasm to define how to manage memory, and
> does not require WASI or any other guest imports."

Memory management is entirely delegated to the guest runtime, whether that's a language's GC,
manual allocation, or stack-based management.

**Source**: [http-wasm.io/http-handler-abi](https://http-wasm.io/http-handler-abi/)

### 9.3 WebAssembly Component Model (Canonical ABI)

**Allocation:**
- Uses `cabi_realloc(old_ptr, old_size, align, new_size) -> new_ptr`
- Canonical ABI function for allocation and reallocation

**Deallocation:**
- Call `cabi_realloc(ptr, old_size, 1, 0)` to free memory
- Deallocation is a special case of reallocation

**Current issues:**
- Cannot pass pre-allocated buffers; host must call guest's allocator
- Inefficient for embedded systems where dynamic allocation is discouraged
- Multiple copies and allocations for simple data transfers

**Proposed optimization** (Issue #314):
- "Caller-supplied buffer hack" using special globals
- Allows guest to detect and reuse pre-allocated buffers
- Implementation in toolchain (wit-bindgen, wasi-libc) rather than spec change

**Source**: [WebAssembly/component-model#314](https://github.com/WebAssembly/component-model/issues/314)

### 9.4 Summary: Why otelwasm Omits `dealloc`

Based on this survey, we concluded that omitting an explicit `dealloc` host function is:

1. **Standard practice**: Both proxy-wasm and http-wasm follow this pattern
2. **Language-neutral**: Accommodates GC languages (Go), RAII languages (Rust), and manual
   management (Zig)
3. **Ownership-clear**: Guest allocates → guest owns → guest deallocates
4. **Simpler ABI**: Fewer host functions, fewer edge cases

### 9.5 Alternative Approaches Considered and Rejected

#### Option 1: Export `dealloc(ptr, size)`

**Pros:**
- Symmetric API (`otelwasm_memory_allocate` + `dealloc`)
- Host could explicitly free memory after processing

**Cons:**
- Incompatible with Go's GC (pinned allocations must be unpinned, not explicitly freed)
- Incompatible with Rust's Drop trait (double-free risk)
- Adds complexity without clear benefit
- Not needed given single-threaded execution model

**Verdict**: Rejected

#### Option 2: Component Model's `cabi_realloc` pattern

**Pros:**
- Standardized approach
- Supports both allocation and deallocation

**Cons:**
- Requires Canonical ABI support in guest SDK
- More complex API surface
- Still has efficiency issues (Component Model Issue #314)
- Overkill for otelwasm's simple data passing needs

**Verdict**: Rejected for v1; may revisit in future versions if Component Model adoption
increases

#### Option 3: Buffer-passing (http-wasm pattern)

**Pros:**
- No dynamic allocation needed
- Very simple ABI

**Cons:**
- Doesn't align with OTel Collector's push-based consumer model
- Requires two-pass protocol (get size, allocate, retry) for variable-size data
- Less ergonomic for plugin authors

**Verdict**: Rejected; the push model better mirrors Collector interfaces

### 9.6 Implications for Guest SDK Implementations

Each language's guest SDK handles memory differently:

**Go:**
- Uses pinning map to prevent GC collection
- `TakeOwnership(ptr, size)` unpins and returns the buffer
- Memory freed by Go's GC after buffer is no longer referenced

**Rust:**
- `otelwasm_memory_allocate` uses `std::alloc::alloc` with appropriate `Layout`
- Guest SDK provides safe wrapper that deallocates in Drop implementation
- No GC concerns; deallocation is deterministic

**Zig:**
- `otelwasm_memory_allocate` uses `std.heap.wasm_allocator`
- Guest SDK manages allocation/deallocation via Zig's allocator interface
- Manual memory management, explicit deallocation

All three approaches work correctly with the guest-managed deallocation pattern.

### 9.7 References

- [proxy-wasm ABI Specification](https://github.com/proxy-wasm/spec)
- [http-wasm HTTP Handler ABI](https://http-wasm.io/http-handler-abi/)
- [WebAssembly Component Model](https://github.com/WebAssembly/component-model)
- [Component Model Issue #314: Efficient memory passing](https://github.com/WebAssembly/component-model/issues/314)
- [A Practical Guide to WebAssembly Memory](https://radu-matei.com/blog/practical-guide-to-wasm-memory/)
- [Wasm needs a better memory management story](https://github.com/WebAssembly/design/issues/1397)
