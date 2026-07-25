// logs_add_attribute_handle_tinygo is the TinyGo counterpart of
// logs_add_attribute_handle: it adds a fixed attribute to every log record
// through the handle/accessor ABI.
//
// The guest/pdatahandle SDK (and the pdata module it depends on) doesn't
// compile under TinyGo, so the few host functions used are imported directly.
// Build with:
//
//	tinygo build -target=wasip1 -buildmode=c-shared -scheduler=none -gc=conservative -no-debug -o main.wasm .
//
// -scheduler=none matters: the default (asyncify-based) scheduler adds a
// large fixed cost to every exported function call (~3ms under the wazero
// interpreter, ~36us compiled).
package main

import "unsafe"

func main() {}

// Field IDs are OTLP protobuf field numbers, shared with the host side.
const (
	fieldLogsResourceLogs      = 1
	fieldResourceLogsScopeLogs = 2
	fieldScopeLogsLogRecords   = 2
	fieldLogRecordAttributes   = 6
)

const telemetryTypeLogs = 1 << 1

//go:wasmimport opentelemetry.io/wasm pdataCurrentLogs
func pdataCurrentLogs() uint32

//go:wasmimport opentelemetry.io/wasm pdataCommitCurrentLogs
func pdataCommitCurrentLogs()

//go:wasmimport opentelemetry.io/wasm pdataGetMsg
func pdataGetMsg(handle, field uint32) uint32

//go:wasmimport opentelemetry.io/wasm pdataSliceLen
func pdataSliceLen(handle uint32) uint32

//go:wasmimport opentelemetry.io/wasm pdataSliceGet
func pdataSliceGet(handle, index uint32) uint32

//go:wasmimport opentelemetry.io/wasm pdataMapPutStr
func pdataMapPutStr(handle, keyPtr, keyLen, valPtr, valLen uint32)

func mapPutStr(h uint32, key, value string) {
	keyPtr := uint32(uintptr(unsafe.Pointer(unsafe.StringData(key))))
	valPtr := uint32(uintptr(unsafe.Pointer(unsafe.StringData(value))))
	pdataMapPutStr(h, keyPtr, uint32(len(key)), valPtr, uint32(len(value)))
}

//go:wasmexport getSupportedTelemetry
func getSupportedTelemetry() uint32 {
	return telemetryTypeLogs
}

//go:wasmexport processLogs
func processLogs() uint32 {
	logs := pdataCurrentLogs()
	rls := pdataGetMsg(logs, fieldLogsResourceLogs)
	for i := uint32(0); i < pdataSliceLen(rls); i++ {
		sls := pdataGetMsg(pdataSliceGet(rls, i), fieldResourceLogsScopeLogs)
		for j := uint32(0); j < pdataSliceLen(sls); j++ {
			lrs := pdataGetMsg(pdataSliceGet(sls, j), fieldScopeLogsLogRecords)
			for k := uint32(0); k < pdataSliceLen(lrs); k++ {
				attrs := pdataGetMsg(pdataSliceGet(lrs, k), fieldLogRecordAttributes)
				mapPutStr(attrs, "example_key", "example_value")
			}
		}
	}
	pdataCommitCurrentLogs()
	return 0
}
