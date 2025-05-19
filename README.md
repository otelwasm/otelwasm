# OTelWasm

Project Status: **Experimental**

This project is a PoC for a WebAssembly (Wasm) based OpenTelemetry Collector plugins. It is not intended for production use, and it may include breaking changes without notice.

## Acknowledgements

This project originally started by Anuraag (Rag) Agrawal (@anuraaga). Most of the code and design is based on [his prior work](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/11772).

This project also leverages the work of the [kube-scheduler-wasm-extension](https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension) project, which is a great example of how to use WebAssembly as a runtime for plugin.

## Architecture

```mermaid
flowchart LR
  R1(wasmreceiver 1) --> P1[wasmprocessor 1]
  R2(wasmreceiver 2) --> P1
  RM(...) ~~~ P1
  RN(wasmreceiver N) --> P1
  P1 --> P2[wasmprocessor 2]
  P2 --> PM[...]
  PM --> PN[wasmprocessor N]
  PN --> FO((fan-out))
  FO --> E1[[wasmexporter 1]]
  FO --> E2[[wasmexporter 2]]
  FO ~~~ EM[[...]]
  FO --> EN[[wasmexporter N]]
```

```mermaid
sequenceDiagram
    participant Collector as OpenTelemetry Collector
    participant Processor as wasmprocessor (Host Side)
    participant Memory as Wasm Memory
    participant GuestModule as Wasm Module (Guest Side)
    
    Collector->>Processor: ConsumeMetrics
    
    Note over Processor: メトリクスデータを処理しWasm呼び出しを準備
    
    Processor->>GuestModule: processMetrics()関数呼び出し
    
    Note over Processor,Memory: プロトコルバッファ処理
    Processor->>Memory: メトリクスをprotobuf形式でWasmメモリに書き込み
    
    Memory-->>GuestModule: メモリからメトリクスデータを読取り
    
    Note over GuestModule: カスタムメトリクス<br>処理ロジックを実行
    
    GuestModule->>Memory: 処理結果をprotobuf形式でWasmメモリに書き込み
    
    Memory-->>Processor: メモリから処理結果を読取り
    
    Note over Processor: protobufからメトリクス形式にデシリアライズ
    
    GuestModule-->>Processor: 処理ステータスを返す
    
    alt 処理成功
        Processor-->>Collector: 処理済みメトリクスを返す
    else エラー発生
        Processor-->>Collector: エラーを返す
    end
    
    Collector->>Collector: パイプライン処理を継続
```
