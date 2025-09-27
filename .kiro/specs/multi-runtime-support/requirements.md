# マルチランタイムサポート要件定義

## 背景

現在のOTelWasmは、Wazeroランタイムにハードコードされた実装となっている。将来的にWasmtime、WasmEdge、その他のランタイムに対応し、WASI Preview 2 (Component Model) への対応準備も必要である。

## 目標

1. **ランタイム抽象化**: 特定のWasmランタイムに依存しない抽象化レイヤーの導入
2. **既存実装の保持**: 現在のWazero実装の動作を完全に保持
3. **設定による選択**: 設定ファイルでランタイムエンジンを選択可能
4. **将来拡張性**: 新しいランタイムの追加が容易な設計
5. **WASI対応準備**: WASI Preview 2対応への基盤整備

## ユースケース

### UC1. デフォルトランタイム使用（既存動作）

**シナリオ**: 既存ユーザーが設定変更なしでWazeroランタイムを使用

```yaml
processors:
  wasm/attributes:
    path: "./examples/processor/attributesprocessor/main.wasm"
    plugin_config:
      actions:
        - key: "environment"
          value: "production"
          action: set
```

**期待結果**: 自動的にWazeroランタイムが選択され、既存と同じ動作

### UC2. 明示的ランタイム選択

**シナリオ**: ユーザーがWasmtimeランタイムを明示的に選択

```yaml
processors:
  wasm/attributes:
    path: "./examples/processor/attributesprocessor/main.wasm"
    runtime_config:
      type: "wasmtime"
    plugin_config:
      actions:
        - key: "environment"
          value: "production"
          action: set
```

**期待結果**: Wasmtimeランタイムが使用される

### UC3. Wazero固有設定

**シナリオ**: Wazeroランタイムの固有オプションを指定

```yaml
processors:
  wasm/attributes:
    path: "./examples/processor/attributesprocessor/main.wasm"
    runtime_config:
      type: "wazero"
      wazero:
        mode: "compiled"
    plugin_config:
      actions:
        - key: "service"
          value: "my-service"
          action: set
```

**期待結果**: Wazeroコンパイルモードで実行

### UC4. Wasmtime固有設定

**シナリオ**: Wasmtimeランタイムの固有オプションを指定

```yaml
processors:
  wasm/attributes:
    path: "./examples/processor/attributesprocessor/main.wasm"
    runtime_config:
      type: "wasmtime"
      wasmtime:
        strategy: "cranelift"
    plugin_config:
      actions:
        - key: "service"
          value: "my-service"
          action: set
```

**期待結果**: WasmtimeのCraneliftエンジンで実行

### UC5. 複数コンポーネントでの異なるランタイム

**シナリオ**: プロセッサーとエクスポーターで異なるランタイムを使用

```yaml
processors:
  wasm/attributes:
    path: "./processor.wasm"
    runtime_config:
      type: "wazero"
      wazero:
        mode: "compiled"
        compilation_cache: true
    plugin_config:
      actions:
        - key: "processed"
          value: "true"
          action: set

exporters:
  wasm/custom:
    path: "./exporter.wasm"
    runtime_config:
      type: "wasmtime"
      wasmtime:
        strategy: "cranelift"
        opt_level: "speed"
    plugin_config:
      endpoint: "http://localhost:8080/metrics"
```

**期待結果**: 各コンポーネントが指定されたランタイムで独立して動作

## 機能要件

### FR1. ランタイム抽象化インターフェース

- `Runtime`, `CompiledModule`, `ModuleInstance`, `FunctionInstance`, `Memory`の5つのコアインターフェースを定義
- Wasmの基本的なライフサイクル（コンパイル、インスタンス化、関数呼び出し、メモリ操作、クローズ）をカバー
- 特定のランタイム実装に依存しない汎用的な設計

#### Runtime インターフェース
- `Compile(ctx context.Context, binary []byte) (CompiledModule, error)`
- `Close(ctx context.Context) error`

#### CompiledModule インターフェース
- `Instantiate(ctx context.Context) (ModuleInstance, error)`
- `Close(ctx context.Context) error`

#### ModuleInstance インターフェース
- `Function(name string) FunctionInstance`
- `Memory() Memory`
- `Close(ctx context.Context) error`

#### FunctionInstance インターフェース
- `Call(ctx context.Context, params ...uint64) ([]uint64, error)`

#### Memory インターフェース
- `Read(offset uint32, size uint32) ([]byte, bool)`
- `Write(offset uint32, data []byte) bool`

### FR2. Wazeroアダプター実装

- 既存のWazero実装を新しいインターフェースに適合させるアダプターを作成
- 現在の機能・性能を完全に保持
- `wazero.Runtime`, `api.Module`, `api.Function`, `api.Memory`を抽象インターフェースでラップ

### FR3. OpenTelemetry設定統合

- OpenTelemetryコンポーネント設定で`runtime_config`セクションをサポート
- `plugin_config`と同じ階層で`runtime_config`を指定可能
- 設定構造体を以下のように拡張:
  - `type`: ランタイムエンジン種別（"wazero", "wasmtime" など）
  - ランタイム固有セクション（`wazero`, `wasmtime`など）でランタイム固有オプションを指定
- `runtime_config`が未指定の場合はWazeroをデフォルト使用
- 不明なランタイムタイプが指定された場合は適切なエラーを返す

### FR4. ランタイム固有設定サポート

- 各ランタイムエンジンが独自の設定項目を持てる構造
- Wazero固有設定:
  - `mode`: 実行モード（"interpreter", "compiled"）- 既存機能
- Wasmtime固有設定（最小限）:
  - `strategy`: コンパイル戦略（"cranelift"など）
- 新しいランタイム追加時に設定項目を独立して定義可能
- ランタイム固有設定の妥当性検証機能

### FR5. ファクトリパターン実装

- 設定に基づいて適切なRuntimeを生成する`newRuntime`関数
- 将来の拡張に対応できるswitch文による分岐
- 各エンジン固有の初期化処理を適切に処理
- ランタイム固有設定の解析と妥当性検証

### FR6. 既存コードの互換性保持

- `WasmPlugin`構造体のAPIは変更しない
- 既存のテストがすべて通ることを保証
- プロセッサー、エクスポーター、レシーバーの動作に影響を与えない

## 非機能要件

### NFR1. 性能要件

- 抽象化レイヤー導入によるオーバーヘッドは最小限に抑制
- 既存のWazero実装と同等の性能を維持
- メモリ使用量の増加は5%以内

### NFR2. 拡張性要件

- 新しいランタイムの追加時のコード変更は最小限
- インターフェースの変更なしに新しいランタイムを追加可能
- WASI Preview 2対応時の基盤として活用可能

### NFR3. 品質要件

- すべてのインターフェースに対するユニットテストを作成
- 既存のテストスイートをすべて通す
- ドキュメント化されたインターフェース設計

## 制約事項

### C1. 後方互換性

- 既存のWazero実装の動作を完全に保持
- 設定ファイルのフォーマット変更は最小限
- 外部APIに破壊的変更を加えない

### C2. 段階的実装

- 第1段階: Wazeroアダプターの実装とリファクタリング
- 第2段階: 他のランタイム実装の追加（本要件の範囲外）
- WASI Preview 2対応は将来の機能拡張として位置づけ

### C3. 設定の単純性

- 設定項目の追加は最小限
- デフォルト動作は現在と同一
- 設定エラー時の適切なフォールバック

## 検収基準

### AC1. インターフェース定義

- [ ] 5つのコアインターフェースが定義されている
- [ ] 各インターフェースのメソッドシグネチャが仕様通り
- [ ] インターフェースのドキュメント化が完了

### AC2. Wazeroアダプター

- [ ] WazeroのすべてのAPIが新しいインターフェースでラップされている
- [ ] 既存のテストがすべて通る
- [ ] パフォーマンスが既存実装と同等

### AC3. 設定機能

- [ ] OpenTelemetry設定でruntime_configセクションがサポートされている
- [ ] plugin_configと同じ階層でruntime_configが指定可能
- [ ] typeフィールドでランタイム種別が正しく解析される
- [ ] ランタイム固有設定（wazero, wasmtimeなど）が正しく解析される
- [ ] runtime_config未指定時にWazeroがデフォルト使用される
- [ ] 不明なランタイムタイプ指定時に適切なエラー

### AC4. ランタイム固有設定

- [ ] Wazero固有設定（mode）が正しく適用される
- [ ] Wasmtime固有設定（strategy）のフレームワークが準備される
- [ ] ランタイム固有設定の妥当性検証が機能する
- [ ] 新しいランタイム追加時の設定拡張が容易

### AC5. ファクトリ機能

- [ ] newRuntime関数が実装されている
- [ ] 設定に基づいて正しいRuntimeが生成される
- [ ] ランタイム固有設定の解析と適用が正しく動作
- [ ] エラーハンドリングが適切

### AC6. 既存機能保持

- [ ] WasmPlugin APIに変更がない
- [ ] プロセッサー、エクスポーター、レシーバーが正常動作
- [ ] すべての既存テストが通る

## 除外事項

- 実際のWasmtime、WasmEdge実装（将来の機能拡張）
- WASI Preview 2の具体的実装
- Component Modelのサポート
- 既存設定の移行ツール作成