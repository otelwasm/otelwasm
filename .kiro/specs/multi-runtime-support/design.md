# マルチランタイムサポート設計書

## 概要

本設計書は、OTelWasmプロジェクトにマルチランタイムサポートを導入するための詳細設計を定義する。現在のWazeroハードコード実装を抽象化し、将来的にWasmtime、WasmEdge等の他ランタイムへの対応を可能にする。

## アーキテクチャ概要

### 現在の構成
```
OTel Component → WasmPlugin → Wazero (直接依存)
```

### 提案する構成
```
OTel Component → WasmPlugin → Runtime Interface → Runtime Adapter → Wazero/Wasmtime/etc
```

## インターフェース設計

### 1. コアインターフェース（拡張版）

新しいパッケージ `runtime` を作成し、Wazero固有部分の隠蔽を含む以下のインターフェースを定義：

```go
// runtime/interfaces.go
package runtime

import "context"

// Runtime represents a Wasm runtime engine
type Runtime interface {
	// Compile compiles the given Wasm binary into a CompiledModule
	Compile(ctx context.Context, binary []byte) (CompiledModule, error)
	// InstantiateWithHost creates module instance with host functions and runtime-specific setup
	InstantiateWithHost(ctx context.Context, module CompiledModule, hostModule HostModule) (ModuleInstance, Context, error)
	// Close closes the runtime and releases all resources
	Close(ctx context.Context) error
}

// CompiledModule represents a compiled Wasm module, ready for instantiation
type CompiledModule interface {
	// Close releases the resources associated with the compiled module
	Close(ctx context.Context) error
}

// ModuleInstance represents an instantiated Wasm module
type ModuleInstance interface {
	// Function returns a handle to an exported function
	// Returns nil if the function is not found
	Function(name string) FunctionInstance
	// Memory returns the memory instance of the module
	// Returns nil if the module does not export memory
	Memory() Memory
	// Close closes the instance and releases its resources
	Close(ctx context.Context) error
}

// FunctionInstance represents an exported function from a Wasm module
type FunctionInstance interface {
	// Call executes the function with the given parameters
	Call(ctx context.Context, params ...uint64) ([]uint64, error)
}

// Memory represents the linear memory of a Wasm module instance
type Memory interface {
	// Read reads 'size' bytes from the memory at 'offset'
	Read(offset uint32, size uint32) ([]byte, bool)
	// Write writes 'data' to the memory at 'offset'
	Write(offset uint32, data []byte) bool
}

// Context holds runtime-specific state (WASI, host modules, etc.)
// This is opaque to WasmPlugin and managed entirely by runtime adapters
type Context interface {
	// Close releases runtime-specific resources
	Close(ctx context.Context) error
}

// HostModule defines host functions to be made available to WASM modules
type HostModule interface {
	// Functions returns the list of host functions to register
	Functions() []HostFunction
}

// HostFunction represents a single host function definition
type HostFunction struct {
	ModuleName   string
	FunctionName string
	Function     interface{} // Runtime-specific function implementation
	ParamTypes   []ValueType
	ResultTypes  []ValueType
}

// ValueType represents WASM value types
type ValueType int

const (
	ValueTypeI32 ValueType = iota
	ValueTypeI64
	ValueTypeF32
	ValueTypeF64
)
```

### 2. エラーハンドリング設計

- Goの標準に従い、失敗可能な操作はすべて`error`を返す
- `Function`と`Memory`の取得は存在しない場合に`nil`を返す
- `context.Context`を活用したタイムアウト・キャンセル処理

## 設定システム設計

### 1. 設定構造体の拡張

```go
// wasmplugin/config.go

// Config defines the common configuration for WASM components
type Config struct {
	Path          string         `mapstructure:"path"`
	PluginConfig  PluginConfig   `mapstructure:"plugin_config"`
	RuntimeConfig *RuntimeConfig `mapstructure:"runtime_config,omitempty"`
}

// Runtime type constants
const (
	RuntimeTypeWazero   = "wazero"
	RuntimeTypeWasmtime = "wasmtime"
)

// RuntimeConfig is the configuration for the WASM plugin runtime
type RuntimeConfig struct {
	// Type specifies the Wasm runtime to use
	Type string `mapstructure:"type"`

	// Runtime-specific configurations
	Wazero   *WazeroConfig   `mapstructure:"wazero,omitempty"`
	Wasmtime *WasmtimeConfig `mapstructure:"wasmtime,omitempty"`

	// Remaining holds unknown runtime configurations for future extensibility
	Remaining map[string]interface{} `mapstructure:",remain"`
}

// WazeroConfig holds wazero-specific configurations
type WazeroConfig struct {
	// Mode is the runtime mode (interpreter or compiled) - existing feature
	Mode WazeroRuntimeMode `mapstructure:"mode,omitempty"`
}

// WasmtimeConfig holds wasmtime-specific configurations (minimal placeholder)
type WasmtimeConfig struct {
	// Strategy specifies compilation strategy
	Strategy WasmtimeStrategy `mapstructure:"strategy,omitempty"`
}

type WazeroRuntimeMode string

const (
	WazeroRuntimeModeInterpreter WazeroRuntimeMode = "interpreter"
	WazeroRuntimeModeCompiled    WazeroRuntimeMode = "compiled"
)

type WasmtimeStrategy string

const (
	WasmtimeStrategyCranelift WasmtimeStrategy = "cranelift"
)
```

### 2. 設定例

```yaml
# Default (Wazero)
processors:
  wasm/attributes:
    path: "./main.wasm"
    plugin_config:
      actions: [...]

# Explicit Wazero with custom settings
processors:
  wasm/attributes:
    path: "./main.wasm"
    runtime_config:
      type: "wazero"
      wazero:
        mode: "compiled"
    plugin_config:
      actions: [...]

# Wasmtime (future)
processors:
  wasm/attributes:
    path: "./main.wasm"
    runtime_config:
      type: "wasmtime"
      wasmtime:
        strategy: "cranelift"
    plugin_config:
      actions: [...]
```

### 3. 妥当性検証

```go
func (c *Config) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("path is required")
	}

	if c.RuntimeConfig != nil {
		return c.RuntimeConfig.Validate()
	}
	return nil
}

func (rc *RuntimeConfig) Validate() error {
	supportedTypes := []string{RuntimeTypeWazero, RuntimeTypeWasmtime}

	if rc.Type != "" {
		for _, supported := range supportedTypes {
			if rc.Type == supported {
				return rc.validateSpecific()
			}
		}
		return fmt.Errorf("unsupported runtime type: %s", rc.Type)
	}
	return nil
}

func (rc *RuntimeConfig) validateSpecific() error {
	switch rc.Type {
	case RuntimeTypeWazero:
		return rc.Wazero.Validate()
	case RuntimeTypeWasmtime:
		return rc.Wasmtime.Validate()
	default:
		return nil
	}
}
```

## ファクトリパターン設計

### 1. ランタイム登録システム

```go
// runtime/factory.go
package runtime

import (
	"fmt"
	"github.com/otelwasm/otelwasm/wasmplugin"
)

// Factory creates a new Runtime instance
type Factory func(config interface{}) (Runtime, error)

var runtimeFactories = make(map[string]Factory)

// Register registers a runtime factory
func Register(name string, factory Factory) {
	if _, exists := runtimeFactories[name]; exists {
		panic(fmt.Sprintf("runtime %s already registered", name))
	}
	runtimeFactories[name] = factory
}

// New creates a new Runtime based on the config
func New(ctx context.Context, config *wasmplugin.RuntimeConfig) (Runtime, error) {
	// Default to wazero if not specified
	runtimeType := config.Type
	if runtimeType == "" {
		runtimeType = RuntimeTypeWazero
	}

	factory, ok := runtimeFactories[runtimeType]
	if !ok {
		return nil, fmt.Errorf("unknown runtime type: %s", runtimeType)
	}

	// Extract runtime-specific configuration
	var specificConfig interface{}
	switch runtimeType {
	case RuntimeTypeWazero:
		specificConfig = config.Wazero
	case RuntimeTypeWasmtime:
		specificConfig = config.Wasmtime
	default:
		specificConfig = config.Remaining[runtimeType]
	}

	return factory(specificConfig)
}

// List returns all registered runtime types
func List() []string {
	types := make([]string, 0, len(runtimeFactories))
	for t := range runtimeFactories {
		types = append(types, t)
	}
	return types
}
```

### 2. 自動登録機構

```go
// runtime/wazero/init.go
package wazero

import (
	"context"
	"github.com/otelwasm/otelwasm/runtime"
	"github.com/otelwasm/otelwasm/wasmplugin"
)

func init() {
	runtime.Register(RuntimeTypeWazero, newWazeroRuntime)
}

func newWazeroRuntime(config interface{}) (runtime.Runtime, error) {
	wazeroConfig, ok := config.(*wasmplugin.WazeroConfig)
	if !ok || wazeroConfig == nil {
		// Use default configuration
		wazeroConfig = &wasmplugin.WazeroConfig{
			Mode: WazeroRuntimeModeInterpreter,
		}
	}

	// Initialize wazero runtime with config
	return &wazeroRuntime{
		config: wazeroConfig,
		// ... other initialization
	}, nil
}
```

## Wazeroアダプター設計

### 1. アダプター構造体

```go
// runtime/wazero/adapter.go
package wazero

import (
	"context"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/otelwasm/otelwasm/runtime"
	"github.com/otelwasm/otelwasm/wasmplugin"
)

type wazeroRuntime struct {
	runtime wazero.Runtime
	config  *wasmplugin.WazeroConfig
}

type wazeroCompiledModule struct {
	module wazero.CompiledModule
}

type wazeroModuleInstance struct {
	instance api.Module
}

type wazeroFunctionInstance struct {
	function api.Function
}

type wazeroMemory struct {
	memory api.Memory
}
```

### 2. インターフェース実装

```go
func (r *wazeroRuntime) Compile(ctx context.Context, binary []byte) (runtime.CompiledModule, error) {
	compiled, err := r.runtime.CompileModule(ctx, binary)
	if err != nil {
		return nil, fmt.Errorf("wazero compile error: %w", err)
	}

	// Validate memory export as per existing logic
	if _, ok := compiled.ExportedMemories()["memory"]; !ok {
		return nil, fmt.Errorf("wasm: guest doesn't export memory[memory]")
	}

	return &wazeroCompiledModule{module: compiled}, nil
}

func (r *wazeroRuntime) Close(ctx context.Context) error {
	return r.runtime.Close(ctx)
}

func (m *wazeroCompiledModule) Instantiate(ctx context.Context) (runtime.ModuleInstance, error) {
	// This requires access to the runtime for instantiation
	// Design decision: How to handle host module instantiation?
	// Option 1: Pass runtime reference to CompiledModule
	// Option 2: Handle host module setup at Runtime level

	instance, err := r.runtime.InstantiateModule(ctx, m.module, wazero.NewModuleConfig())
	if err != nil {
		return nil, fmt.Errorf("wazero instantiate error: %w", err)
	}

	return &wazeroModuleInstance{instance: instance}, nil
}

func (m *wazeroModuleInstance) Function(name string) runtime.FunctionInstance {
	fn := m.instance.ExportedFunction(name)
	if fn == nil {
		return nil
	}
	return &wazeroFunctionInstance{function: fn}
}

func (m *wazeroModuleInstance) Memory() runtime.Memory {
	memory := m.instance.Memory()
	if memory == nil {
		return nil
	}
	return &wazeroMemory{memory: memory}
}

func (f *wazeroFunctionInstance) Call(ctx context.Context, params ...uint64) ([]uint64, error) {
	return f.function.Call(ctx, params...)
}

func (mem *wazeroMemory) Read(offset uint32, size uint32) ([]byte, bool) {
	return mem.memory.Read(offset, size)
}

func (mem *wazeroMemory) Write(offset uint32, data []byte) bool {
	return mem.memory.Write(offset, data)
}
```

## WasmPlugin統合設計

### 1. WasmPlugin構造体の変更（Wazero固有部分の隠蔽）

**課題**: 現在の`WasmPlugin`構造体にはWazero固有のフィールドが含まれている：
- `Sys wasi.System` (wasi-go固有)
- `wasiP1HostModule *wasi_snapshot_preview1.Module` (Wazero WASI固有)

**解決策**: ランタイム固有の詳細をアダプター内に隠蔽する

```go
// wasmplugin/plugin.go

type WasmPlugin struct {
	// Runtime is the WebAssembly runtime (abstracted)
	Runtime runtime.Runtime

	// Module is the instantiated WASM module (abstracted)
	Module runtime.ModuleInstance

	// PluginConfigJSON is the JSON representation of the plugin config
	PluginConfigJSON []byte

	// ExportedFunctions from the WASM module (abstracted)
	ExportedFunctions map[string]runtime.FunctionInstance

	// runtimeContext holds runtime-specific state (opaque to WasmPlugin)
	// This includes WASI state, host module instances, and other runtime-specific data
	runtimeContext runtime.Context
}
```

### 2. NewWasmPlugin関数の変更（ランタイム抽象化対応）

```go
func NewWasmPlugin(ctx context.Context, cfg *Config, requiredFunctions []string) (*WasmPlugin, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Set defaults
	cfg.Default()

	f, err := os.Open(cfg.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// Use new runtime factory
	runtime, err := runtime.New(ctx, cfg.RuntimeConfig)
	if err != nil {
		return nil, fmt.Errorf("runtime initialization failed: %w", err)
	}

	// Compile the module
	compiled, err := runtime.Compile(ctx, bytes)
	if err != nil {
		runtime.Close(ctx)
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	// Create host module definition
	hostModule := createHostModule()

	// Instantiate with host functions and runtime-specific setup (WASI, etc.)
	// This replaces all the manual WASI and host module setup
	mod, runtimeCtx, err := runtime.InstantiateWithHost(ctx, compiled, hostModule)
	if err != nil {
		compiled.Close(ctx)
		runtime.Close(ctx)
		return nil, fmt.Errorf("module instantiation failed: %w", err)
	}

	// Validate required functions
	exportedFunctions := make(map[string]runtime.FunctionInstance)
	allRequiredFunctions := append(requiredFunctions, builtInGuestFunctions...)
	for _, funcName := range allRequiredFunctions {
		fn := mod.Function(funcName)
		if fn == nil {
			return nil, fmt.Errorf("required function %s not exported: %w", funcName, ErrRequiredFunctionNotExported)
		}
		exportedFunctions[funcName] = fn
	}

	// Convert plugin config to JSON
	pluginConfigJSON, err := json.Marshal(cfg.PluginConfig)
	if err != nil {
		return nil, fmt.Errorf("plugin config marshalling failed: %w", err)
	}

	return &WasmPlugin{
		Runtime:           runtime,
		Module:            mod,
		PluginConfigJSON:  pluginConfigJSON,
		ExportedFunctions: exportedFunctions,
		runtimeContext:    runtimeCtx,
	}, nil
}

// createHostModule creates the host module definition
// This replaces the current instantiateHostModule function
func createHostModule() runtime.HostModule {
	return &otelHostModule{
		functions: []runtime.HostFunction{
			{
				ModuleName:   "opentelemetry.io/wasm",
				FunctionName: "currentTraces",
				Function:     &HostFunctionImpl{Name: "currentTraces"},
				ParamTypes:   []runtime.ValueType{runtime.ValueTypeI32, runtime.ValueTypeI32},
				ResultTypes:  []runtime.ValueType{runtime.ValueTypeI32},
			},
			{
				ModuleName:   "opentelemetry.io/wasm",
				FunctionName: "currentMetrics",
				Function:     &HostFunctionImpl{Name: "currentMetrics"},
				ParamTypes:   []runtime.ValueType{runtime.ValueTypeI32, runtime.ValueTypeI32},
				ResultTypes:  []runtime.ValueType{runtime.ValueTypeI32},
			},
			// ... 他のホスト関数も同様に定義
		},
	}
}

type otelHostModule struct {
	functions []runtime.HostFunction
}

func (h *otelHostModule) Functions() []runtime.HostFunction {
	return h.functions
}

// HostFunctionImpl provides a runtime-agnostic representation of host functions
// The actual implementation will be provided by the runtime adapter
type HostFunctionImpl struct {
	Name string
}

// GetImplementation returns the runtime-specific implementation
// This method will be called by runtime adapters to get their specific function implementation
func (h *HostFunctionImpl) GetImplementation(runtimeType string) interface{} {
	switch h.Name {
	case "currentTraces":
		return getHostFunction(runtimeType, "currentTraces")
	case "currentMetrics":
		return getHostFunction(runtimeType, "currentMetrics")
	// ... 他の関数も同様
	default:
		return nil
	}
}

// getHostFunction returns the runtime-specific host function implementation
// This function will be implemented in each runtime adapter package
func getHostFunction(runtimeType, functionName string) interface{} {
	// This will be implemented by registering runtime-specific implementations
	// For example: wazeroHostFunctions[functionName] or wasmtimeHostFunctions[functionName]
	return nil // placeholder
}
```

## 課題と解決策

### 1. WASI統合の複雑性

**問題**: 現在のWASI実装がWazero固有のAPIに依存している

**解決策**:
- WASI初期化をランタイム抽象化レイヤーの外で処理
- 必要に応じてランタイム固有のWASI統合を各アダプター内で実装

### 2. ホストモジュール統合

**問題**: ホスト関数の登録がWazero固有

**解決策**:
- ホスト関数定義を抽象化
- 各ランタイムアダプターでホスト関数を登録する仕組みを提供

### 3. パフォーマンス最適化

**問題**: インターフェース経由でのオーバーヘッド

**解決策**:
- インライン最適化の活用
- ホットパスでの直接アクセス（必要に応じて）
- ベンチマークによる定量評価

## テスト戦略

### 1. ユニットテスト

- 各インターフェースの実装に対するテスト
- ファクトリパターンの動作テスト
- 設定解析・妥当性検証のテスト

### 2. 統合テスト

- 既存のWasmプラグインテストを新実装で実行
- パフォーマンス回帰テスト
- エラーハンドリングテスト

### 3. 互換性テスト

- 既存設定ファイルとの互換性確認
- デフォルト動作の確認

## 実装順序

1. **runtime パッケージの作成**
   - インターフェース定義
   - ファクトリパターン実装

2. **設定システムの拡張**
   - RuntimeConfig 構造体の実装
   - 妥当性検証の追加

3. **Wazero アダプターの実装**
   - 既存機能の完全移植
   - パフォーマンステスト

4. **WasmPlugin の統合**
   - 既存APIの互換性保持
   - エラーハンドリングの改善

5. **テストとドキュメント**
   - 包括的テストスイート
   - 移行ガイドの作成

## 将来拡張への準備

### 1. 新ランタイム追加手順

1. `runtime/<new-runtime>` パッケージ作成
2. アダプター構造体とインターフェース実装
3. `init()` 関数でファクトリ登録
4. 設定構造体の追加（`RuntimeConfig`）
5. テストの実装

### 2. WASI Preview 2 対応準備

- Component Model インターフェースの追加定義
- ランタイム能力の段階的確認機構
- 設定での機能有効化フラグ

この設計により、既存機能を保持しつつ、将来の拡張に対応できる柔軟で堅牢なマルチランタイムサポートが実現される。