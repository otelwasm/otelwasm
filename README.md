# OTelWasm

Project Status: **Experimental**

This project is a PoC for a WebAssembly (Wasm) based OpenTelemetry Collector plugins. It is not intended for production use, and it may include breaking changes without notice.

## WebAssembly runtimes

The WASM plugins can run on one of the following runtimes, selectable per component with the `runtime.name` configuration:

- `wazero` (default): a pure-Go runtime with no external dependencies. `runtime.mode` selects `interpreter` (default) or the experimental `compiled` mode.
- `wasmedge`: the [WasmEdge](https://wasmedge.org/) runtime through its CGO bindings. `runtime.mode` selects `interpreter` (default) or `compiled`, which AOT-compiles the module at startup (this can take minutes for large modules).

```yaml
processors:
  wasm:
    path: "path/to/plugin.wasm"
    runtime:
      name: wasmedge
      mode: compiled # optional: AOT-compile at startup
```

To avoid the AOT compilation cost at startup, the module can also be precompiled into a universal wasm binary with the WasmEdge CLI and used directly with the default `interpreter` mode (the embedded native code is used automatically):

```console
$ wasmedge compile --optimize 2 plugin.wasm plugin-aot.wasm
```

WasmEdge support is opt-in at build time. It requires the WasmEdge shared library (`make install-wasmedge` installs it to `~/.wasmedge`) and building with the `wasmedge` build tag with CGO enabled:

```console
$ make install-wasmedge
$ make otelwasmcol-wasmedge  # build the collector with WasmEdge support
$ make test-wasmedge         # run tests against the WasmEdge runtime
```

Binaries built without the `wasmedge` build tag return an error when `runtime.name: wasmedge` is configured.

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
