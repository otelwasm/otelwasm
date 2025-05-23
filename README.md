# OTelWasm

Project Status: **Experimental**

This project is a PoC for a WebAssembly (Wasm) based OpenTelemetry Collector plugins. It is not intended for production use, and it may include breaking changes without notice.

## Build OTelWasm OTel Collector distribution

If you want to build OTelWasm OTel Collector distribution, execute the following command.

```shell
make otelwasmcol
```

This command generates Go project files of OTel Collector and build otelwasmcol binary that combines multiple components coming from OTel Collector Core distribution with wasm-ready components provided by OTelWasm. The otelwasmcol binary is generated in `bin` directory in the project root path.

## Build example guest wasm binaries

Example wasm components are in `examples`, and you can build them at once by the following command.

```shell
make build-wasm-examples
```

Each wasm binary is generated under each directory, for example, the wasm version of `attributesprocessor` is generated at `examples/processor/attributes/processor/main.wasm`.

## How to run wasm-powered OTel Collector

After building example wasm binaries and otelwasmcol itself, now you're ready to try.

Here's example otel-collector config to work with OTelWasm.

```yaml
receivers:
  wasm/otlpreceiver:
    # Currently, otlpreceiver only accepts OTLP/HTTP because of otelwasm bug.
    # You can't use OTLP/gRPC at the moment.
    # https://github.com/otelwasm/otelwasm/issues/59
    path: "./examples/receiver/otlpreceiver/main.wasm"
processors:
  wasm/attributes:
    path: "./examples/processor/attributesprocessor/main.wasm"
    plugin_config:
      # Accepting same config as upstream attributesprocessor
      # https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/attributesprocessor
      actions:
      - key: inserted-attributes-by-wasm
        action: insert
        value: hello-from-wasm
exporters:
  wasm/otlphttpexporter:
    path: "./examples/exporter/otlphttpexporter/main.wasm"
    plugin_config:
      # Accepting same config as upstream otlphttpexporter 
      # https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter/otlphttpexporter
      endpoint: "http://localhost:4319"
      # compression and sending_queue should be set to the following values due to otelwasm bug.
      # https://github.com/otelwasm/otelwasm/issues/60
      compression: none
      sending_queue:
        enabled: false

service:
  pipelines:
    traces:
      receivers: [wasm/otlpreceiver]
      processors: [wasm/attributes]
      exporters: [wasm/otlphttpexporter]
```

After saving the config as `config.yaml`, you can try otelwasmcol by the following command.

```shell
./bin/otelwasmcol_darwin_arm64 --config ./config.yaml
```

## Acknowledgements

This project originally started by Anuraag (Rag) Agrawal (@anuraaga). Most of the code and design is based on [his prior work](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/11772).

This project also leverages the work of the [kube-scheduler-wasm-extension](https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension) project, which is a great example of how to use WebAssembly as a runtime for plugin.
