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

## Loading plugins from an OCI registry

Wasm plugins can be distributed as OCI images. Set an `oci://` reference as the plugin path and the collector pulls the module at startup, using credentials from your local docker config (`docker login`):

```yaml
processors:
  wasm/attributes:
    path: "oci://ghcr.io/otelwasm/attributesprocessor:latest"
```

To publish a plugin, build the `wasmpush` CLI and push a compiled wasm module:

```shell
make wasmpush
./bin/wasmpush -metadata metadata.json examples/processor/attributesprocessor/main.wasm ghcr.io/otelwasm/attributesprocessor:latest
```

By default the plugin is pushed as an OCI artifact with otelwasm media types (`application/vnd.otelwasm.plugin.content.layer.v1+wasm` and `application/vnd.otelwasm.plugin.metadata.v1+json`). For registries that don't support OCI artifacts, pass `-format compat` to push a standard container image whose single tar.gz layer contains `plugin.wasm` (compatible with [solo-io's wasm image spec](https://github.com/solo-io/wasm/blob/master/spec/spec-compat.md)). Pulling handles both formats transparently.

### Local registry end-to-end test

Start a local registry, build the binaries, and push the example receiver and exporter:

```shell
docker run --rm -d --name otelwasm-registry -p 127.0.0.1:5000:5000 registry:2
make build-wasm-examples wasmpush otelwasmcol
./bin/wasmpush examples/receiver/otlpreceiver/main.wasm localhost:5000/otelwasm/otlpreceiver:e2e
./bin/wasmpush examples/exporter/stdout/main.wasm localhost:5000/otelwasm/stdoutexporter:e2e
```

Save the following Collector manifest as `config-oci.yaml`:

```yaml
receivers:
  wasm/otlp:
    path: "oci://localhost:5000/otelwasm/otlpreceiver:e2e"

exporters:
  wasm/stdout:
    path: "oci://localhost:5000/otelwasm/stdoutexporter:e2e"

service:
  pipelines:
    traces:
      receivers: [wasm/otlp]
      exporters: [wasm/stdout]
```

Start the Collector:

```shell
./bin/otelwasmcol_$(go env GOOS)_$(go env GOARCH) --config ./config-oci.yaml
```

In another terminal, send a trace over OTLP/HTTP:

```shell
curl --fail --header 'Content-Type: application/json' \
  --data-binary '{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"oci-e2e"}}]},"scopeSpans":[{"scope":{"name":"otelwasm-e2e"},"spans":[{"traceId":"5b8efff798038103d269b633813fc60c","spanId":"eee19b7ec3c1b174","name":"oci-plugin-roundtrip","kind":1,"startTimeUnixNano":"1750000000000000000","endTimeUnixNano":"1750000001000000000","status":{"code":1}}]}]}]}' \
  http://localhost:4318/v1/traces
```

The request should return HTTP 200 and the Collector should print the `oci-plugin-roundtrip` span. Stop the Collector, then remove the registry with `docker stop otelwasm-registry`.

## Acknowledgements

This project originally started by Anuraag (Rag) Agrawal (@anuraaga). Most of the code and design is based on [his prior work](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/11772).

This project also leverages the work of the [kube-scheduler-wasm-extension](https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension) project, which is a great example of how to use WebAssembly as a runtime for plugin.
