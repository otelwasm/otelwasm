//go:build wasm

package pdatahandle

import "github.com/otelwasm/otelwasm/guest/internal/mem"

//go:wasmimport opentelemetry.io/wasm pdataCurrentTraces
func pdataCurrentTraces() uint32

//go:wasmimport opentelemetry.io/wasm pdataCommitCurrentTraces
func pdataCommitCurrentTraces()

//go:wasmimport opentelemetry.io/wasm pdataCurrentMetrics
func pdataCurrentMetrics() uint32

//go:wasmimport opentelemetry.io/wasm pdataCommitCurrentMetrics
func pdataCommitCurrentMetrics()

//go:wasmimport opentelemetry.io/wasm pdataCurrentLogs
func pdataCurrentLogs() uint32

//go:wasmimport opentelemetry.io/wasm pdataCommitCurrentLogs
func pdataCommitCurrentLogs()

//go:wasmimport opentelemetry.io/wasm pdataGetI64
func pdataGetI64(handle, field uint32) int64

//go:wasmimport opentelemetry.io/wasm pdataSetI64
func pdataSetI64(handle, field uint32, value int64)

//go:wasmimport opentelemetry.io/wasm pdataGetF64
func pdataGetF64(handle, field uint32) float64

//go:wasmimport opentelemetry.io/wasm pdataSetF64
func pdataSetF64(handle, field uint32, value float64)

//go:wasmimport opentelemetry.io/wasm pdataGetStr
func pdataGetStr(handle, field, buf uint32, bufLimit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm pdataSetStr
func pdataSetStr(handle, field, ptr, size uint32)

//go:wasmimport opentelemetry.io/wasm pdataGetMsg
func pdataGetMsg(handle, field uint32) uint32

//go:wasmimport opentelemetry.io/wasm pdataSliceLen
func pdataSliceLen(handle uint32) uint32

//go:wasmimport opentelemetry.io/wasm pdataSliceGet
func pdataSliceGet(handle, index uint32) uint32

//go:wasmimport opentelemetry.io/wasm pdataMapLen
func pdataMapLen(handle uint32) uint32

//go:wasmimport opentelemetry.io/wasm pdataMapGetStr
func pdataMapGetStr(handle, keyPtr, keyLen, buf uint32, bufLimit mem.BufLimit) (len uint32)

//go:wasmimport opentelemetry.io/wasm pdataMapPutStr
func pdataMapPutStr(handle, keyPtr, keyLen, valPtr, valLen uint32)
