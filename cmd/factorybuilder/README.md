# factorybuilder
* Build existing OpenTelemetry Collector components into wasm files for otelwasm

## Installation
```console
$ go install github.com/otelwasm/otelwasm/cmd/factorybuilder@latest
```

## Usage
* Build attributesprocessor

```
$ factorybuilder -o main.wasm github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor
```
