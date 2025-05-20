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
    participant Processor as wasmprocessor (Host Side)
    participant GuestModule as Wasm Module (Guest Side)

    Note over Processor: テレメトリーデータが入力される

    Processor->>GuestModule: Guest Function呼び出し

    Note over Processor: Go StructからOTLP protobuf形式にシリアライズ
    Processor->>GuestModule: テレメトリーをOTLP protobuf形式でWasmメモリに書き込み

    Note over GuestModule: OTLP protobuf形式からデシリアライズ

    GuestModule->>GuestModule: 処理を実行

    Note over GuestModule: OTLP protobuf形式にシリアライズ

    GuestModule-->>Processor: Wasmメモリから処理結果を読取り

    Note over Processor: OTLP protobuf形式からGo Structにデシリアライズ

    GuestModule-->>Processor: 処理ステータスを返す

    Note over Processor: 次のコンポーネントに処理を渡す
```

### wasmreceiver

```mermaid
sequenceDiagram
    participant Processor as wasmprocessor (Host Side)
    participant GuestModule as Wasm Module (Guest Side)

    Note over Processor: テレメトリーデータが入力される

    Processor->>GuestModule: Guest Function呼び出し

    Note over Processor: Go StructからOTLP protobuf形式にシリアライズ
    Processor->>GuestModule: テレメトリーをOTLP protobuf形式でWasmメモリに書き込み

    Note over GuestModule: OTLP protobuf形式からデシリアライズ

    GuestModule->>GuestModule: 処理を実行

    Note over GuestModule: OTLP protobuf形式にシリアライズ

    GuestModule-->>Processor: Wasmメモリから処理結果を読取り

    Note over Processor: OTLP protobuf形式からGo Structにデシリアライズ

    GuestModule-->>Processor: 処理ステータスを返す

    Note over Processor: 次のコンポーネントに処理を渡す
```

### wasmexporter
