# otelwasm Runtime Benchmarks

Date: 2026-07-25
Branch: `worktree-wasmedge-runtime` (WasmEdge runtime support + PR [#90](https://github.com/otelwasm/otelwasm/pull/90) handle/accessor pdata ABI)

## Environment

| Item | Value |
|---|---|
| Machine | Apple M3 Pro (11 cores), macOS (Darwin 24.3.0) |
| Host Go | go 1.26.4 (guests built with go 1.24.2 via wasibuilder) |
| wazero | v1.12.0 |
| WasmEdge | 0.14.1 (WasmEdge-go v0.14.0, CGO) |
| pdata | v1.31.0 |

Runtime configurations measured:

- **native Go** — the upstream processor linked directly into the collector (no wasm)
- **wazero interpreter** — pure-Go runtime, default configuration (`runtime: {name: wazero, mode: interpreter}`)
- **wazero compiled** — wazero compiler (JIT) backend (`mode: compiled`)
- **WasmEdge interpreter** — WasmEdge via CGO (`runtime: {name: wasmedge}`)
- **WasmEdge AOT** — guest precompiled with `wasmedge compile --optimize 2` (universal wasm binary)

All numbers are medians of repeated `go test -bench` runs (`-benchmem`), single-threaded pipeline calls.

## Workload 1: attributesprocessor logs pipeline (proto ABI)

A realistic enrichment + masking pipeline through the wasm-wrapped
`attributesprocessor` (collector-contrib) guest, proto-serialization ABI.

- Payload: 100 log records / batch (10 services x 10 records, typical resource / HTTP / user attributes, ~33 KB serialized)
- Actions: 2 static upserts, SHA-1 hash of `user.email`, regex extract on `http.url`
- Benchmark: `BenchmarkLogsAttributesProcessor*` in `wasmprocessor`

| Configuration | time/batch | vs native | B/op | allocs/op |
|---|---:|---:|---:|---:|
| native Go | 70 µs | 1x | 42 KB | 1,004 |
| wazero interpreter | 118.6 ms | ~1,700x | 6.4 MB | 120,578 |
| wazero compiled | 2.42 ms | ~35x | 367 KB | 4,869 |
| WasmEdge interpreter | 168.2 ms | ~2,400x | 275 KB | 4,881 |
| WasmEdge AOT | 1.18 ms | ~17x | 266 KB | 4,879 |

Observations:

- Enabling JIT/AOT closes the interpreter's ~100x gap: WasmEdge AOT is ~150x faster than its interpreter, wazero compiled ~50x faster than its interpreter.
- On the proto ABI, WasmEdge AOT is the fastest wasm configuration (~2x faster than wazero compiled).
- The remaining ~17x gap vs native is dominated by the protobuf round-trip of the whole batch on every call (see Workload 2).
- wazero's interpreter allocates heavily (pure-Go interpretation of the guest); WasmEdge's memory overhead is small even in interpreter mode.

## Workload 2: handle/accessor ABI vs proto ABI (PR #90), logs

Identical guest workload in two ABI variants: walk every log record and
upsert one string attribute per record.

- **native** — the same walk + upsert as a plain Go processor through the same processorhelper harness (`BenchmarkHandleVsProtoLogsNative`)
- `logs_add_attribute_proto` — proto ABI: the whole batch is serialized to the guest and the result serialized back
- `logs_add_attribute_handle` — handle ABI (PR #90): the guest reads/mutates host-held pdata through fine-grained host functions (no bulk serialization)
- Benchmark: `BenchmarkHandleVsProtoLogs` (wazero), `BenchmarkHandleVsProtoLogsWasmEdge` (WasmEdge), medians of 3 runs (native: 5 runs)

### time per batch

| Configuration | ABI | 1 record | 100 records | 1000 records |
|---|---|---:|---:|---:|
| **native Go** | — | **90 ns** | **2.17 µs** | **21.3 µs** |
| wazero interpreter | proto | 430 µs | 27.0 ms | 294.7 ms |
| wazero interpreter | handle | 6.6 µs | 185 µs | 1.82 ms |
| wazero compiled | proto | 17.0 µs | 769 µs | 8.23 ms |
| **wazero compiled** | **handle** | **1.66 µs** | **29.1 µs** | **273 µs** |
| WasmEdge interpreter | proto | 688 µs | 47.6 ms | 523 ms |
| WasmEdge interpreter | handle | 15.2 µs | 441 µs | 4.32 ms |
| WasmEdge AOT | proto | 9.6 µs | 419 µs | 4.13 ms |
| WasmEdge AOT | handle | 11.2 µs | 270 µs | 2.66 ms |

### slowdown vs native (100 records)

| Configuration | proto | handle |
|---|---:|---:|
| wazero interpreter | 12,400x | 85x |
| wazero compiled | 354x | **13x** |
| WasmEdge interpreter | 21,900x | 203x |
| WasmEdge AOT | 193x | 124x |

### handle ABI speedup over proto ABI (100 records)

| Configuration | proto | handle | speedup |
|---|---:|---:|---:|
| wazero interpreter | 26.9 ms | 185 µs | 145x |
| wazero compiled | 769 µs | 29.1 µs | 26x |
| WasmEdge interpreter | 47.6 ms | 441 µs | 108x |
| WasmEdge AOT | 419 µs | 270 µs | 1.6x |

### memory (100 records)

| Configuration | ABI | B/op | allocs/op |
|---|---|---:|---:|
| native Go | — | 1.7 KB | 104 |
| wazero interpreter | proto | 1.43 MB | 33,982 |
| wazero interpreter | handle | 40.5 KB | 1,459 |
| wazero compiled | proto | 163 KB | 2,431 |
| wazero compiled | handle | 29.6 KB | 525 |
| WasmEdge interpreter | proto | 114 KB | 2,441 |
| WasmEdge interpreter | handle | 104.7 KB | 3,916 |
| WasmEdge AOT | proto | 113 KB | 2,441 |
| WasmEdge AOT | handle | 104.7 KB | 3,916 |

Observations:

- **wazero compiled + handle ABI is the fastest wasm configuration** (29 µs / 100 records), ~9x faster than WasmEdge AOT with the handle ABI and ~14x faster than the best proto configuration — and only **13x slower than native** (proto's best is 193x).
- The native baseline is extremely cheap here (a map upsert per record), so the wasm slowdown factors are an upper bound; heavier per-record logic would shrink the relative gap.
- **The handle ABI's benefit is strongly runtime-dependent.** On wazero, a host function call is a plain Go call, so the many fine-grained `pdata*` calls are nearly free (145x speedup on the interpreter). On WasmEdge every host call crosses the CGO boundary with per-call boxing in the bindings (43k allocs/op at 1000 records vs 5k on wazero), which caps the gain at 1.6x — and at 1 record the handle ABI is even slightly slower than proto (11.2 µs vs 9.6 µs).
- Consequently the optimal runtime flips with the ABI: **proto ABI → WasmEdge AOT; handle ABI → wazero compiled.**
- The WasmEdge handle numbers include some overhead from the wazero-convention adapter (`adaptWazeroHostFn`), but the dominant cost is the CGO round-trip per host call in the wasmedge-go bindings.

## Workload 3: guest language comparison (handle ABI, logs)

The identical handle-ABI workload (walk every log record, upsert one string
attribute) implemented in three guest languages:

- **go** — Go 1.24 `c-shared` via the `guest/pdatahandle` SDK (the guest from Workload 2)
- **tinygo** — TinyGo 0.41, direct `//go:wasmimport` declarations (the SDK's pdata dependency doesn't compile under TinyGo), built with `-scheduler=none -gc=conservative -no-debug`
- **rust** — Rust (wasm32-wasip1, cdylib), direct `extern "C"` imports, `opt-level=3` + LTO

Benchmark: `BenchmarkHandleLogsGuestFlavors` (wazero) / `BenchmarkHandleLogsGuestFlavorsWasmEdge` (WasmEdge), medians of 3 runs.

### binary size and compile cost

| Guest | wasm size | WasmEdge AOT compile (O2) |
|---|---:|---:|
| go | 27 MB | ~7 min |
| tinygo | 10.1 KB | 0.7 s |
| rust | 647 B | 0.06 s |

### time per batch (100 records)

| Runtime | go | tinygo | rust |
|---|---:|---:|---:|
| wazero interpreter | 183.9 µs | 31.0 µs | **27.6 µs** |
| wazero compiled | 27.8 µs | 24.3 µs | **20.1 µs** |
| WasmEdge interpreter | 422 µs | 248 µs | **204 µs** |
| WasmEdge AOT | 262 µs | 237 µs | **194 µs** |

### time per batch (1000 records)

| Runtime | go | tinygo | rust |
|---|---:|---:|---:|
| wazero interpreter | 1.87 ms | 283 µs | **248 µs** |
| wazero compiled | 272 µs | 220 µs | **184 µs** |
| WasmEdge interpreter | 4.22 ms | 2.44 ms | **1.92 ms** |
| WasmEdge AOT | 2.62 ms | 2.33 ms | **1.86 ms** |

Observations:

- **Rust is the fastest guest in every configuration**; Rust + wazero compiled (20.1 µs / 100 records) is the fastest wasm result measured in this document, ~9x from native.
- **Lean guests transform the wazero interpreter**: go 184 µs → rust 27.6 µs (6.7x) at 100 records, because the interpreter no longer executes the Go runtime's bookkeeping. A zero-dependency, pure-Go deployment (wazero interpreter + Rust/TinyGo guest) gets within ~40% of the JIT.
- On WasmEdge the guest language matters much less — the per-host-call CGO cost dominates (all flavors make the same ~500 host calls per 100-record batch).
- **TinyGo pitfall**: the default (asyncify-based) scheduler adds a large fixed cost to every exported call — ~3 ms under the wazero interpreter, ~36 µs compiled. `-scheduler=none` removes it (this workload needs no goroutines). `-gc=conservative` vs `-gc=leaking` made no measurable difference here.
- TinyGo/Rust guests also make WasmEdge AOT practical to generate at build time (sub-second vs minutes for big-Go guests).

## Startup / compilation costs

| Item | Cost |
|---|---|
| wazero interpreter instantiation (27–45 MB guest) | ~1–1.5 s |
| wazero compiled JIT at startup (45 MB guest) | ~10 s |
| WasmEdge AOT compile, O2 (27–28 MB guest) | ~5–7 min |
| WasmEdge AOT compile, O2 (45 MB guest) | ~25 min |
| WasmEdge AOT binary size (45 MB guest) | 78 MB |

WasmEdge AOT compilation can run at startup (`runtime: {name: wasmedge, mode: compiled}`) or offline (`wasmedge compile --optimize 2 in.wasm out.wasm`, then point `path` at the output — the runtime loads guests from the file path so the embedded native code is used automatically).

## Notes

- The wazero compiler backend crash with the attributesprocessor guest on arm64 ([#64](https://github.com/otelwasm/otelwasm/issues/64)) no longer reproduces on wazero >= v1.11; these runs used v1.12.0.
- During this work a host-side protocol bug was found and fixed: payloads larger than the guest's initial 2048-byte read buffer were returned as size 0 instead of the required size, so any batch over ~2 KB crashed the guest (`writeBytesIfUnderLimit`). All benchmarks above include the fix.
- Native Go in Workload 1 is not directly comparable to Workload 2 (different processing pipelines); Workload 2 has its own native baseline running the identical walk + upsert.

## Reproduction

```console
$ make install-wasmedge          # WasmEdge shared library into ~/.wasmedge
$ make test-wasmedge             # tests on the WasmEdge runtime
$ make benchmark-wasmedge        # precompiles AOT guests, then runs all benchmarks
```

Or manually:

```console
$ export CGO_CFLAGS="-I$HOME/.wasmedge/include"
$ export CGO_LDFLAGS="-L$HOME/.wasmedge/lib -lwasmedge -Wl,-rpath,$HOME/.wasmedge/lib"
$ cd wasmprocessor
$ go test -tags wasmedge -run '^$' -bench 'BenchmarkLogsAttributesProcessor' -benchmem -count=5 .
$ go test -tags wasmedge -run '^$' -bench 'BenchmarkHandleVsProtoLogs' -benchmem -count=3 .   # includes the native baseline
$ make build-wasm-guest-flavors   # TinyGo / Rust guests (requires tinygo, rust wasm32-wasip1)
$ go test -tags wasmedge -run '^$' -bench 'BenchmarkHandleLogsGuestFlavors' -benchmem -count=3 .
```
