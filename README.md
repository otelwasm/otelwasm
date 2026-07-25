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

## Acknowledgements

This project originally started by Anuraag (Rag) Agrawal (@anuraaga). Most of the code and design is based on [his prior work](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/11772).

This project also leverages the work of the [kube-scheduler-wasm-extension](https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension) project, which is a great example of how to use WebAssembly as a runtime for plugin.
