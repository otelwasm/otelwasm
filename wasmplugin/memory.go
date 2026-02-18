package wasmplugin

import (
	"github.com/tetratelabs/wazero/api"
)

// These utility functions are derived from the kube-scheduler-wasm-extension.
// https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension

// writeBytesIfUnderLimit writes bytes to memory if they fit within the limit
func writeBytesIfUnderLimit(memory api.Memory, bytes []byte, buf, bufLimit uint32) uint32 {
	if uint32(len(bytes)) > bufLimit {
		return 0
	}
	if !memory.Write(buf, bytes) {
		return 0
	}
	return uint32(len(bytes))
}
