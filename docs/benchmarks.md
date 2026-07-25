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

- `logs_add_attribute_proto` — proto ABI: the whole batch is serialized to the guest and the result serialized back
- `logs_add_attribute_handle` — handle ABI (PR #90): the guest reads/mutates host-held pdata through fine-grained host functions (no bulk serialization)
- Benchmark: `BenchmarkHandleVsProtoLogs` (wazero), `BenchmarkHandleVsProtoLogsWasmEdge` (WasmEdge), medians of 3 runs

### time per batch

| Configuration | ABI | 1 record | 100 records | 1000 records |
|---|---|---:|---:|---:|
| wazero interpreter | proto | 430 µs | 27.0 ms | 294.7 ms |
| wazero interpreter | handle | 6.6 µs | 185 µs | 1.82 ms |
| wazero compiled | proto | 17.0 µs | 769 µs | 8.23 ms |
| **wazero compiled** | **handle** | **1.66 µs** | **29.1 µs** | **273 µs** |
| WasmEdge interpreter | proto | 688 µs | 47.6 ms | 523 ms |
| WasmEdge interpreter | handle | 15.2 µs | 441 µs | 4.32 ms |
| WasmEdge AOT | proto | 9.6 µs | 419 µs | 4.13 ms |
| WasmEdge AOT | handle | 11.2 µs | 270 µs | 2.66 ms |

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
| wazero interpreter | proto | 1.43 MB | 33,982 |
| wazero interpreter | handle | 40.5 KB | 1,459 |
| wazero compiled | proto | 163 KB | 2,431 |
| wazero compiled | handle | 29.6 KB | 525 |
| WasmEdge interpreter | proto | 114 KB | 2,441 |
| WasmEdge interpreter | handle | 104.7 KB | 3,916 |
| WasmEdge AOT | proto | 113 KB | 2,441 |
| WasmEdge AOT | handle | 104.7 KB | 3,916 |

Observations:

- **wazero compiled + handle ABI is the fastest configuration overall** (29 µs / 100 records), ~9x faster than WasmEdge AOT with the handle ABI and ~14x faster than the best proto configuration.
- **The handle ABI's benefit is strongly runtime-dependent.** On wazero, a host function call is a plain Go call, so the many fine-grained `pdata*` calls are nearly free (145x speedup on the interpreter). On WasmEdge every host call crosses the CGO boundary with per-call boxing in the bindings (43k allocs/op at 1000 records vs 5k on wazero), which caps the gain at 1.6x — and at 1 record the handle ABI is even slightly slower than proto (11.2 µs vs 9.6 µs).
- Consequently the optimal runtime flips with the ABI: **proto ABI → WasmEdge AOT; handle ABI → wazero compiled.**
- The WasmEdge handle numbers include some overhead from the wazero-convention adapter (`adaptWazeroHostFn`), but the dominant cost is the CGO round-trip per host call in the wasmedge-go bindings.

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
- Native Go in Workload 1 is not directly comparable to Workload 2 (different processing pipelines).

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
$ go test -tags wasmedge -run '^$' -bench 'BenchmarkHandleVsProtoLogs' -benchmem -count=3 .
```
