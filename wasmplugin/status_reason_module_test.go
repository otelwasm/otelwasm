package wasmplugin

// buildStatusReasonConsumeTracesModule returns a minimal ABI v1 module used by
// negative tests. The module sets status reason through host import and returns
// non-zero status from otelwasm_consume_traces.
func buildStatusReasonConsumeTracesModule(reason string) []byte {
	module := []byte{
		0x00, 0x61, 0x73, 0x6d, // magic
		0x01, 0x00, 0x00, 0x00, // version
	}

	appendSection := func(sectionID byte, payload []byte) {
		module = append(module, sectionID)
		module = append(module, encodeULEB128Test(uint32(len(payload)))...)
		module = append(module, payload...)
	}

	// Type section:
	// 0: (i32, i32) -> ()      [set_status_reason import]
	// 1: (i32) -> i32          [otelwasm_memory_allocate]
	// 2: (i32, i32) -> i32     [otelwasm_consume_traces]
	// 3: () -> ()              [abi marker / _initialize]
	// 4: () -> i32             [get_supported_telemetry]
	appendSection(0x01, []byte{
		0x05, // 5 types
		0x60, 0x02, 0x7f, 0x7f, 0x00,
		0x60, 0x01, 0x7f, 0x01, 0x7f,
		0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f,
		0x60, 0x00, 0x00,
		0x60, 0x00, 0x01, 0x7f,
	})

	// Import section: import opentelemetry.io/wasm.set_status_reason as function index 0.
	importPayload := []byte{0x01}
	importPayload = append(importPayload, encodeULEB128Test(uint32(len(otelWasm)))...)
	importPayload = append(importPayload, otelWasm...)
	importPayload = append(importPayload, encodeULEB128Test(uint32(len(setStatusReason)))...)
	importPayload = append(importPayload, setStatusReason...)
	importPayload = append(importPayload, 0x00, 0x00) // kind=func, type index 0
	appendSection(0x02, importPayload)

	// Function section: 5 local functions (indices 1..5).
	appendSection(0x03, []byte{
		0x05, // 5 functions
		0x01, // otelwasm_memory_allocate
		0x02, // otelwasm_consume_traces
		0x03, // otelwasm_abi_version_0_1_0
		0x04, // get_supported_telemetry
		0x03, // _initialize
	})

	// Memory section: one memory, min 1 page.
	appendSection(0x05, []byte{
		0x01, // 1 memory
		0x00, // min only
		0x01, // min pages
	})

	// Export section.
	// Function indices include imports, so local functions start at 1.
	exportPayload := []byte{0x06}
	exportPayload = append(exportPayload, encodeULEB128Test(uint32(len(guestExportMemory)))...)
	exportPayload = append(exportPayload, guestExportMemory...)
	exportPayload = append(exportPayload, 0x02, 0x00) // memory index 0
	exportPayload = appendExportedFunc(exportPayload, memoryAllocateFunction, 1)
	exportPayload = appendExportedFunc(exportPayload, consumeTracesFunction, 2)
	exportPayload = appendExportedFunc(exportPayload, abiVersionV1MarkerExport, 3)
	exportPayload = appendExportedFunc(exportPayload, getSupportedTelemetry, 4)
	exportPayload = appendExportedFunc(exportPayload, "_initialize", 5)
	appendSection(0x07, exportPayload)

	// Code section.
	// otelwasm_memory_allocate(size) -> i32.const 4096
	allocBody := []byte{0x00, 0x41}
	allocBody = append(allocBody, encodeULEB128Test(4096)...)
	allocBody = append(allocBody, 0x0b)

	// otelwasm_consume_traces(data_ptr, data_size):
	//   call set_status_reason(reason_offset=32, reason_len=len(reason))
	//   return ERROR(1)
	consumeBody := []byte{
		0x00,       // local decl count
		0x41, 0x20, // i32.const 32
		0x41, byte(len(reason)), // i32.const reason_len (len(reason) < 128 in tests)
		0x10, 0x00, // call function index 0 (imported set_status_reason)
		0x41, 0x01, // i32.const 1 (ERROR)
		0x0b, // end
	}

	abiMarkerBody := []byte{0x00, 0x0b}
	getSupportedBody := []byte{
		0x00,
		0x41, byte(telemetryTypeTraces),
		0x0b,
	}
	initializeBody := []byte{0x00, 0x0b}

	codePayload := []byte{0x05}
	for _, body := range [][]byte{
		allocBody,
		consumeBody,
		abiMarkerBody,
		getSupportedBody,
		initializeBody,
	} {
		codePayload = append(codePayload, encodeULEB128Test(uint32(len(body)))...)
		codePayload = append(codePayload, body...)
	}
	appendSection(0x0a, codePayload)

	// Data section: write the status reason into guest memory at offset 32.
	dataPayload := []byte{
		0x01,       // 1 segment
		0x00,       // active segment for memory index 0
		0x41, 0x20, // i32.const 32
		0x0b, // end
	}
	dataPayload = append(dataPayload, encodeULEB128Test(uint32(len(reason)))...)
	dataPayload = append(dataPayload, reason...)
	appendSection(0x0b, dataPayload)

	return module
}

func appendExportedFunc(payload []byte, name string, funcIndex byte) []byte {
	payload = append(payload, encodeULEB128Test(uint32(len(name)))...)
	payload = append(payload, name...)
	payload = append(payload, 0x00, funcIndex) // kind=func, function index
	return payload
}
