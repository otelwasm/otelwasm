# マルチランタイムサポート実装計画

## 概要

本計画書は、設計フェーズで確定したマルチランタイムサポートの実装を、段階的かつ安全に進めるための詳細計画を定義する。

## 実装戦略

### 基本方針

1. **段階的実装**: 一度に大きな変更を加えず、検証可能な小さなステップに分割
2. **既存機能保持**: すべての段階で既存テストがパスすることを絶対条件
3. **リスク最小化**: 各フェーズで十分なテストと検証を実施
4. **後方互換性**: 外部APIの変更を避け、内部実装のみを変更

## 実装フェーズ

### フェーズ 1: 基盤整備（1-2日）

**目標**: インターフェース定義と準備作業

#### 1.1 文字列定数化
- [ ] `wasmplugin/config.go`の文字列定数をconstに変更
- [ ] 既存の`RuntimeMode`を`WazeroRuntimeMode`にリネーム
- [ ] `RuntimeTypeWazero`, `RuntimeTypeWasmtime`等の定数を追加

**ファイル影響範囲**:
- `wasmplugin/config.go`

**テスト**:
- 既存のconfig関連テストがすべてパス
- 定数値が正しく設定されていることを確認

#### 1.2 runtimeパッケージ作成
- [ ] `runtime/`ディレクトリとパッケージを作成
- [ ] 7つのコアインターフェースを定義
- [ ] `ValueType`定数と`HostFunction`構造体を定義
- [ ] エラー定義を追加

**作成ファイル**:
```
runtime/
├── interfaces.go      # 7つのインターフェース定義
├── types.go          # ValueType, HostFunction等
├── errors.go         # 共通エラー定義
└── factory.go        # ファクトリ機能
```

**テスト**:
- インターフェースが正しく定義されていることを確認
- 型安全性のテスト

### フェーズ 2: Wazeroアダプター実装（2-3日）

**目標**: 既存Wazero実装を新インターフェースに適合

#### 2.1 Wazeroアダプター構造体作成
- [ ] `runtime/wazero/`パッケージを作成
- [ ] 7つのアダプター構造体を実装
- [ ] Wazero固有の初期化ロジックを実装

**作成ファイル**:
```
runtime/wazero/
├── adapter.go        # wazeroRuntime等の実装
├── host.go          # ホスト関数実装
├── config.go        # Wazero固有設定
└── init.go          # ファクトリ登録
```

**実装内容**:
```go
type wazeroRuntime struct {
    runtime wazero.Runtime
    config  *WazeroConfig
}

type wazeroModuleInstance struct {
    instance api.Module
}

type wazeroContext struct {
    sys              wasi.System
    wasiP1HostModule *wasi_snapshot_preview1.Module
}
```

**テスト**:
- アダプター単体テストを作成
- 既存WasmPluginとの比較テスト

#### 2.2 ホスト関数の抽象化
- [ ] `HostFunctionImpl`構造体を実装
- [ ] ランタイム固有実装の登録機構を作成
- [ ] Wazero用ホスト関数を実装

**テスト**:
- ホスト関数が正しく登録されることを確認
- 関数呼び出しのテスト

### フェーズ 3: WasmPlugin統合（3-4日）

**目標**: WasmPluginを新インターフェースに移行

#### 3.1 WasmPlugin構造体変更
- [ ] Wazero固有フィールドを削除
- [ ] 抽象インターフェースフィールドに置換
- [ ] `runtimeContext`フィールドを追加

**変更内容**:
```go
// 変更前
type WasmPlugin struct {
    Runtime           wazero.Runtime
    Sys               wasi.System
    Module            api.Module
    wasiP1HostModule  *wasi_snapshot_preview1.Module
    // ...
}

// 変更後
type WasmPlugin struct {
    Runtime           runtime.Runtime
    Module            runtime.ModuleInstance
    runtimeContext    runtime.Context
    // ...
}
```

**テスト**:
- 構造体変更後のコンパイルテスト
- フィールドアクセスのテスト

#### 3.2 NewWasmPlugin関数リファクタリング
- [ ] `runtime.New()`を使用するように変更
- [ ] `InstantiateWithHost()`の実装と統合
- [ ] エラーハンドリングの統一

**重要な変更**:
- WASI設定ロジックをアダプター内に移動
- ホストモジュール設定の抽象化
- 複雑な初期化処理の簡素化

**テスト**:
- 初期化処理の正常系テスト
- エラーケースのテスト
- メモリリークテスト

#### 3.3 ProcessFunctionCall統合
- [ ] 抽象インターフェース経由での関数呼び出しに変更
- [ ] コンテキスト管理の統合
- [ ] スタック管理の抽象化

**テスト**:
- 関数呼び出しの正常動作テスト
- パフォーマンス回帰テスト

### フェーズ 4: 各コンポーネント統合（2-3日）

**目標**: processor/exporter/receiverの統合

#### 4.1 wasmprocessor統合
- [ ] 新しいWasmPluginとの統合
- [ ] 設定読み込みの更新
- [ ] テスト実行と検証

#### 4.2 wasmexporter統合
- [ ] 新しいWasmPluginとの統合
- [ ] 設定読み込みの更新
- [ ] テスト実行と検証

#### 4.3 wasmreceiver統合
- [ ] 新しいWasmPluginとの統合
- [ ] 設定読み込みの更新
- [ ] テスト実行と検証

**テスト**:
- 各コンポーネントの既存テストがすべてパス
- 統合テストの実行
- エンドツーエンドテスト

### フェーズ 5: 最終検証・最適化（1-2日）

**目標**: 全体検証とドキュメント更新

#### 5.1 全体テスト
- [ ] 全モジュールのテスト実行
- [ ] ベンチマークテスト実行
- [ ] パフォーマンス回帰の確認

#### 5.2 ドキュメント更新
- [ ] 設計ドキュメントの最終更新
- [ ] 実装に関するドキュメント作成
- [ ] 使用例の更新

#### 5.3 最適化
- [ ] パフォーマンス問題があれば最適化
- [ ] メモリ使用量の最適化
- [ ] エラーメッセージの改善

## 品質保証

### テスト戦略

#### 既存テスト保持
- すべてのフェーズで既存テストがパスすることを確認
- テスト失敗時は即座に原因調査と修正

#### 新規テスト追加
```
runtime/
├── interfaces_test.go
├── factory_test.go
└── wazero/
    ├── adapter_test.go
    ├── host_test.go
    └── integration_test.go

wasmplugin/
├── plugin_test.go        # 新API用テスト
└── compatibility_test.go # 後方互換性テスト
```

#### ベンチマークテスト
- 各フェーズでベンチマーク実行
- 5%以上の性能劣化は許容しない
- メモリアロケーション増加の監視

### エラーハンドリング

#### 共通エラー定義
```go
// runtime/errors.go
var (
    ErrRuntimeNotFound       = errors.New("runtime not found")
    ErrModuleCompileFailed   = errors.New("module compilation failed")
    ErrModuleInstantiateFailed = errors.New("module instantiation failed")
    ErrFunctionNotExported   = errors.New("function not exported")
    ErrInvalidConfiguration  = errors.New("invalid configuration")
)
```

#### エラーラップ戦略
- すべてのエラーに適切なコンテキストを付与
- `fmt.Errorf("step: %w", err)`形式で統一
- ログ出力時の情報量を最大化

### パフォーマンス監視

#### 監視項目
- [ ] 関数呼び出しオーバーヘッド
- [ ] メモリアロケーション回数
- [ ] 初期化時間
- [ ] モジュールロード時間

#### 許容基準
- 既存実装との性能差: ±5%以内
- メモリ使用量増加: +10%以内
- 初期化時間増加: +20%以内

## リスク管理

### 高リスク項目

#### WASI統合の複雑性
**リスク**: WASI設定の移行で機能が失われる可能性
**対策**:
- 段階的移行とテスト
- 既存WASI機能の詳細な検証
- fallback機構の準備

#### ホスト関数の抽象化
**リスク**: ホスト関数の動作が変わる可能性
**対策**:
- 既存ホスト関数の完全再現
- 関数シグネチャの厳密な検証
- 呼び出し結果の比較テスト

#### パフォーマンス回帰
**リスク**: インターフェース経由でのオーバーヘッド
**対策**:
- 各段階でのベンチマーク実行
- ホットパスの最適化
- 必要に応じた直接アクセス保持

### 中リスク項目

#### 設定互換性
**リスク**: 設定形式の変更でユーザー影響
**対策**:
- 既存設定の完全サポート
- 移行パスの提供
- デフォルト値の維持

#### テストカバレッジ
**リスク**: 新しいコードパスのテスト不足
**対策**:
- カバレッジ測定の実施
- 重要パスの手動テスト
- エラーケースの網羅的テスト

## 実装タイムライン

### 全体スケジュール（10-12日）

```
Week 1:
├── Day 1-2: フェーズ1（基盤整備）
├── Day 3-5: フェーズ2（Wazeroアダプター）
└── Day 6-7: フェーズ3開始（WasmPlugin統合）

Week 2:
├── Day 8-9: フェーズ3完了
├── Day 10-11: フェーズ4（コンポーネント統合）
└── Day 12: フェーズ5（最終検証）
```

### マイルストーン

- **M1** (Day 2): インターフェース定義完了
- **M2** (Day 5): Wazeroアダプター完成
- **M3** (Day 9): WasmPlugin統合完了
- **M4** (Day 11): 全コンポーネント統合完了
- **M5** (Day 12): 最終リリース準備完了

### デイリー進捗管理

各日の終了時に以下を確認：
- [ ] 計画タスクの完了状況
- [ ] テスト実行結果
- [ ] 発見された問題と対応状況
- [ ] 翌日の作業計画

## 完了判定基準

### 必須条件
- [ ] すべての既存テストがパス
- [ ] 新規追加テストがすべてパス
- [ ] ベンチマーク結果が許容範囲内
- [ ] コードレビュー完了
- [ ] ドキュメント更新完了

### 品質条件
- [ ] コードカバレッジ80%以上
- [ ] 静的解析エラー0件
- [ ] メモリリーク検出なし
- [ ] 競合状態検出なし

### 運用条件
- [ ] エラーメッセージの適切性確認
- [ ] ログ出力の適切性確認
- [ ] 設定例の動作確認
- [ ] トラブルシューティング手順確認

この実装計画に従って、安全かつ確実にマルチランタイムサポートを実装していきます。