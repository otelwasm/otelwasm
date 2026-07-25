//go:build wasmedge

package wasmplugin

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

// wasmEdgeRuntime is the guestRuntime implementation backed by the WasmEdge
// runtime through its CGO bindings (github.com/second-state/WasmEdge-go).
// It is only available when the binary is built with the "wasmedge" build tag
// and the WasmEdge shared library is installed on the system.
type wasmEdgeRuntime struct {
	// mu serializes guest function calls. A single WasmEdge VM cannot run
	// multiple executions concurrently, and host functions receive the
	// current Stack through the runtime instance instead of a context.
	mu sync.Mutex

	vm         *wasmedge.VM
	conf       *wasmedge.Configure
	hostModule *wasmedge.Module

	// stack is bound for the duration of a guest call so that host
	// functions can access it. Guarded by mu.
	stack *Stack

	// functions is the set of function names exported by the guest module
	functions map[string]struct{}

	closed bool
}

var _ guestRuntime = (*wasmEdgeRuntime)(nil)

// wasmEdgeMemory adapts wasmedge.Memory to the Memory interface.
type wasmEdgeMemory struct {
	mem *wasmedge.Memory
}

func (m wasmEdgeMemory) Read(offset, byteCount uint32) ([]byte, bool) {
	data, err := m.mem.GetData(uint(offset), uint(byteCount))
	if err != nil {
		return nil, false
	}
	return data, true
}

func (m wasmEdgeMemory) Write(offset uint32, v []byte) bool {
	if len(v) == 0 {
		return true
	}
	return m.mem.SetData(v, uint(offset), uint(len(v))) == nil
}

// newWasmEdgeRuntime initializes a WasmEdge VM and instantiates the guest module.
func newWasmEdgeRuntime(ctx context.Context, guestBin []byte, rc RuntimeConfig) (guestRuntime, error) {
	conf := wasmedge.NewConfigure(wasmedge.WASI)
	vm := wasmedge.NewVMWithConfig(conf)

	r := &wasmEdgeRuntime{
		vm:   vm,
		conf: conf,
	}

	fail := func(err error) (guestRuntime, error) {
		r.release()
		return nil, err
	}

	// Initialize WASI. WasmEdge natively implements the WASI socket
	// extension ("wasmedgev2") that guests built for otelwasm expect.
	wasiModule := vm.GetImportModule(wasmedge.WASI)
	if wasiModule == nil {
		return fail(fmt.Errorf("wasm: error retrieving wasi module from wasmedge vm"))
	}
	wasiModule.InitWasi(nil, os.Environ(), nil)

	// Register the otelwasm host module
	hostModule, err := r.buildHostModule()
	if err != nil {
		return fail(fmt.Errorf("wasm: error building host module: %w", err))
	}
	r.hostModule = hostModule
	if err := vm.RegisterModule(hostModule); err != nil {
		return fail(fmt.Errorf("wasm: error registering host module: %w", err))
	}

	// Load, validate and instantiate the guest module
	if err := vm.LoadWasmBuffer(guestBin); err != nil {
		return fail(fmt.Errorf("wasm: error loading guest: %w", err))
	}
	if err := vm.Validate(); err != nil {
		return fail(fmt.Errorf("wasm: error validating guest: %w", err))
	}
	if err := vm.Instantiate(); err != nil {
		return fail(fmt.Errorf("wasm: error instantiating guest: %w", err))
	}

	// Check if the guest exports memory section.
	// As of WebAssembly Core Specification 2.0, there can be at most one memory.
	// https://webassembly.github.io/spec/core/syntax/modules.html#memories
	active := vm.GetActiveModule()
	if active == nil || active.FindMemory(guestExportMemory) == nil {
		return fail(fmt.Errorf("wasm: guest doesn't export memory[%s]", guestExportMemory))
	}

	// Record the exported functions of the guest module
	names, _ := vm.GetFunctionList()
	r.functions = make(map[string]struct{}, len(names))
	for _, name := range names {
		r.functions[name] = struct{}{}
	}

	// Call _initialize as the guest is built as a reactor module
	if _, ok := r.functions["_initialize"]; ok {
		if _, err := vm.Execute("_initialize"); err != nil {
			return fail(fmt.Errorf("wasm: error initializing guest: %w", err))
		}
	}

	return r, nil
}

// HasFunction implements guestRuntime.
func (r *wasmEdgeRuntime) HasFunction(name string) bool {
	_, ok := r.functions[name]
	return ok
}

// CallFunction implements guestRuntime.
func (r *wasmEdgeRuntime) CallFunction(ctx context.Context, name string, stack *Stack) ([]uint64, error) {
	if !r.HasFunction(name) {
		return nil, fmt.Errorf("wasm: function not found: %s", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, fmt.Errorf("wasm: runtime is closed")
	}

	// Bind the stack for host functions during the call
	r.stack = stack
	defer func() { r.stack = nil }()

	returns, err := r.vm.Execute(name)
	if err != nil {
		return nil, fmt.Errorf("wasm: error calling %s: %w", name, err)
	}

	res := make([]uint64, 0, len(returns))
	for _, v := range returns {
		switch value := v.(type) {
		case int32:
			res = append(res, uint64(uint32(value)))
		case int64:
			res = append(res, uint64(value))
		case float32:
			res = append(res, uint64(math.Float32bits(value)))
		case float64:
			res = append(res, math.Float64bits(value))
		default:
			return nil, fmt.Errorf("wasm: unsupported return value type %T from %s", v, name)
		}
	}

	return res, nil
}

// Close implements guestRuntime.
func (r *wasmEdgeRuntime) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true
	r.release()
	return nil
}

func (r *wasmEdgeRuntime) release() {
	if r.vm != nil {
		r.vm.Release()
		r.vm = nil
	}
	if r.hostModule != nil {
		r.hostModule.Release()
		r.hostModule = nil
	}
	if r.conf != nil {
		r.conf.Release()
		r.conf = nil
	}
}

// hostFuncKind describes the shape of an otelwasm host function.
type hostFuncKind int

const (
	// (buf, buf_limit/buf_len i32) -> (i32)
	hostFuncBufIn1Out hostFuncKind = iota
	// (buf, buf_len i32) -> ()
	hostFuncBufIn0Out
	// () -> (i32)
	hostFuncNoIn1Out
)

// buildHostModule creates the otelwasm host module with all host functions
// registered, mirroring the wazero host module definition.
func (r *wasmEdgeRuntime) buildHostModule() (*wasmedge.Module, error) {
	mod := wasmedge.NewModule(otelWasm)
	if mod == nil {
		return nil, fmt.Errorf("failed to create module %s", otelWasm)
	}

	hostFuncs := []struct {
		name string
		kind hostFuncKind
		fn   wasmedgeHostFunc
	}{
		{currentTraces, hostFuncBufIn1Out, r.adaptBufIn1Out(hostCurrentTraces)},
		{currentMetrics, hostFuncBufIn1Out, r.adaptBufIn1Out(hostCurrentMetrics)},
		{currentLogs, hostFuncBufIn1Out, r.adaptBufIn1Out(hostCurrentLogs)},
		{getPluginConfig, hostFuncBufIn1Out, r.adaptBufIn1Out(hostGetPluginConfig)},
		{setResultTraces, hostFuncBufIn0Out, r.adaptBufIn0Out(hostSetResultTraces)},
		{setResultMetrics, hostFuncBufIn0Out, r.adaptBufIn0Out(hostSetResultMetrics)},
		{setResultLogs, hostFuncBufIn0Out, r.adaptBufIn0Out(hostSetResultLogs)},
		{setResultStatusReason, hostFuncBufIn0Out, r.adaptBufIn0Out(hostSetResultStatusReason)},
		{getShutdownRequested, hostFuncNoIn1Out, r.adaptNoIn1Out(hostGetShutdownRequested)},
	}

	for _, hf := range hostFuncs {
		var ftype *wasmedge.FunctionType
		switch hf.kind {
		case hostFuncBufIn1Out:
			ftype = wasmedge.NewFunctionType(
				[]*wasmedge.ValType{wasmedge.NewValTypeI32(), wasmedge.NewValTypeI32()},
				[]*wasmedge.ValType{wasmedge.NewValTypeI32()},
			)
		case hostFuncBufIn0Out:
			ftype = wasmedge.NewFunctionType(
				[]*wasmedge.ValType{wasmedge.NewValTypeI32(), wasmedge.NewValTypeI32()},
				[]*wasmedge.ValType{},
			)
		case hostFuncNoIn1Out:
			ftype = wasmedge.NewFunctionType(
				[]*wasmedge.ValType{},
				[]*wasmedge.ValType{wasmedge.NewValTypeI32()},
			)
		}

		fn := wasmedge.NewFunction(ftype, hf.fn, nil, 0)
		ftype.Release()
		if fn == nil {
			mod.Release()
			return nil, fmt.Errorf("failed to create host function %s", hf.name)
		}
		mod.AddFunction(hf.name, fn)
	}

	return mod, nil
}

type wasmedgeHostFunc = func(data interface{}, callframe *wasmedge.CallingFrame, params []interface{}) ([]interface{}, wasmedge.Result)

// guestMemory returns the guest memory accessible from the calling frame.
func guestMemory(callframe *wasmedge.CallingFrame) (Memory, bool) {
	mem := callframe.GetMemoryByIndex(0)
	if mem == nil {
		return nil, false
	}
	return wasmEdgeMemory{mem: mem}, true
}

// adaptBufIn1Out adapts a shared host function taking (buf, buf_limit) and
// returning a length to the WasmEdge host function calling convention.
// NOTE: Panics must not cross the CGO callback boundary, so they are
// recovered and converted to a trap (Result_Fail).
func (r *wasmEdgeRuntime) adaptBufIn1Out(impl func(*Stack, Memory, uint32, uint32) uint32) wasmedgeHostFunc {
	return func(_ interface{}, callframe *wasmedge.CallingFrame, params []interface{}) (res []interface{}, result wasmedge.Result) {
		defer func() {
			if recovered := recover(); recovered != nil {
				fmt.Fprintf(os.Stderr, "wasm: host function panicked: %v\n", recovered)
				res, result = nil, wasmedge.Result_Fail
			}
		}()

		mem, ok := guestMemory(callframe)
		if !ok {
			return nil, wasmedge.Result_Fail
		}

		buf := uint32(params[0].(int32))
		bufLimit := uint32(params[1].(int32))

		n := impl(r.stack, mem, buf, bufLimit)
		return []interface{}{int32(n)}, wasmedge.Result_Success
	}
}

// adaptBufIn0Out adapts a shared host function taking (buf, buf_len) and
// returning nothing to the WasmEdge host function calling convention.
func (r *wasmEdgeRuntime) adaptBufIn0Out(impl func(*Stack, Memory, uint32, uint32)) wasmedgeHostFunc {
	return func(_ interface{}, callframe *wasmedge.CallingFrame, params []interface{}) (res []interface{}, result wasmedge.Result) {
		defer func() {
			if recovered := recover(); recovered != nil {
				fmt.Fprintf(os.Stderr, "wasm: host function panicked: %v\n", recovered)
				res, result = nil, wasmedge.Result_Fail
			}
		}()

		mem, ok := guestMemory(callframe)
		if !ok {
			return nil, wasmedge.Result_Fail
		}

		buf := uint32(params[0].(int32))
		size := uint32(params[1].(int32))

		impl(r.stack, mem, buf, size)
		return []interface{}{}, wasmedge.Result_Success
	}
}

// adaptNoIn1Out adapts a shared host function taking no memory arguments and
// returning a single value to the WasmEdge host function calling convention.
func (r *wasmEdgeRuntime) adaptNoIn1Out(impl func(*Stack) uint32) wasmedgeHostFunc {
	return func(_ interface{}, callframe *wasmedge.CallingFrame, params []interface{}) (res []interface{}, result wasmedge.Result) {
		defer func() {
			if recovered := recover(); recovered != nil {
				fmt.Fprintf(os.Stderr, "wasm: host function panicked: %v\n", recovered)
				res, result = nil, wasmedge.Result_Fail
			}
		}()

		return []interface{}{int32(impl(r.stack))}, wasmedge.Result_Success
	}
}
