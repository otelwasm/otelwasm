//go:build !wasm

package pdatahandle

import "github.com/otelwasm/otelwasm/guest/internal/mem"

// This file is used to stub out the imports for running tests.

func pdataCurrentTraces() uint32 { return 0 }

func pdataCommitCurrentTraces() {}

func pdataGetI64(handle, field uint32) int64 { return 0 }

func pdataSetI64(handle, field uint32, value int64) {}

func pdataGetStr(handle, field, buf uint32, bufLimit mem.BufLimit) (len uint32) { return }

func pdataSetStr(handle, field, ptr, size uint32) {}

func pdataGetMsg(handle, field uint32) uint32 { return 0 }

func pdataSliceLen(handle uint32) uint32 { return 0 }

func pdataSliceGet(handle, index uint32) uint32 { return 0 }

func pdataMapLen(handle uint32) uint32 { return 0 }

func pdataMapGetStr(handle, keyPtr, keyLen, buf uint32, bufLimit mem.BufLimit) (len uint32) { return }

func pdataMapPutStr(handle, keyPtr, keyLen, valPtr, valLen uint32) {}
