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

### wasmprocessor

```mermaid
sequenceDiagram
    participant Collector as OpenTelemetry Collector
    participant Processor as wasmprocessor (Host Side)
    participant Memory as Wasm Memory
    participant GuestModule as Wasm Module (Guest Side)

    Collector->>Processor: Consume()

    Processor->>GuestModule: Guest Function呼び出し

    Note over Processor: Go StructからOTLP protobuf形式にシリアライズ
    Processor->>Memory: テレメトリーをOTLP protobuf形式でWasmメモリに書き込み

    Memory-->>GuestModule: メモリからテレメトリーデータを読取り

    Note over GuestModule: OTLP protobuf形式からデリシアライズ

    GuestModule->>GuestModule: 処理を実行

    Note over GuestModule: OTLP protobuf形式にシリアライズ

    GuestModule->>Memory: 処理結果をOTLP protobuf形式でWasmメモリに書き込み

    Memory-->>Processor: メモリから処理結果を読取り

    Note over Processor: OTLP protobuf形式からGo Structにデシリアライズ

    GuestModule-->>Processor: 処理ステータスを返す

    alt 処理成功
        Processor-->>Collector: 処理済みのテレメトリーデータを返す
    else エラー発生
        Processor-->>Collector: エラーを返す
    end

    Collector->>Collector: パイプライン処理を継続
```

### wasmreceiver

### wasmexporter
