# OpAMP demo: remotely managing otelwasmcol

This example demonstrates managing `otelwasmcol` via
[OpAMP](https://opentelemetry.io/docs/specs/opamp/) (Open Agent Management
Protocol). The collector runs a pipeline composed entirely of WASM plugins,
and the whole pipeline configuration — including which WASM modules to load
and their `plugin_config` — is distributed and updated remotely by an OpAMP
server.

```
┌──────────────┐  OpAMP (ws)  ┌──────────────────┐  manages   ┌──────────────────────────────┐
│ OpAMP server │◄────────────►│ OpAMP Supervisor │───────────►│ otelwasmcol                  │
│ (server/)    │ remote config└──────────────────┘ start/stop │  wasm receiver (otlp)        │
└──────────────┘                                              │  wasm processor (add attr)   │
      ▲ watches                                               │  wasm exporter (stdout)      │
 remote-config.yaml                                           └──────────────────────────────┘
```

Components:

- `server/` — a minimal OpAMP server built with
  [opamp-go](https://github.com/open-telemetry/opamp-go). It serves
  `remote-config.yaml` as the OpAMP remote config and pushes updates to
  connected agents whenever the file changes.
- `supervisor.yaml` — configuration for the
  [OpAMP Supervisor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/opampsupervisor),
  which connects to the OpAMP server, receives the remote config, and manages
  the `otelwasmcol` process. `otelwasmcol` includes the `opampextension` so
  the supervisor can track its health and effective config.
- `remote-config.yaml` — the collector configuration distributed via OpAMP:
  an all-WASM traces pipeline (OTLP receiver → add_new_attribute processor →
  stdout exporter).

## Prerequisites

Build the collector and the WASM examples from the repository root:

```console
$ make otelwasmcol           # produces bin/otelwasmcol_<os>_<arch>
$ make build-wasm-examples   # produces examples/**/main.wasm
```

Download the OpAMP Supervisor binary (adjust OS/arch as needed):

```console
$ gh release download cmd/opampsupervisor/v0.126.0 \
    -R open-telemetry/opentelemetry-collector-releases \
    -p 'opampsupervisor_0.126.0_darwin_arm64' -O bin/opampsupervisor
$ chmod +x bin/opampsupervisor
```

If `agent::executable` in `supervisor.yaml` does not match your platform
(default: `otelwasmcol_darwin_arm64`), adjust it.

## Running the demo

Terminal 1 — start the OpAMP server:

```console
$ cd examples/opamp/server
$ go run . -config ../remote-config.yaml
opamp-server: ... listening on ws://127.0.0.1:4320/v1/opamp, serving remote config from ../remote-config.yaml
```

Terminal 2 — start the supervisor (which starts `otelwasmcol`).
`OTELWASM_ROOT` is used inside `remote-config.yaml` to locate the WASM
modules:

```console
$ cd examples/opamp
$ export OTELWASM_ROOT=$(git rev-parse --show-toplevel)
$ ../../bin/opampsupervisor --config supervisor.yaml
```

The supervisor connects to the OpAMP server, receives the remote config, and
launches the collector with the all-WASM pipeline. You should see the WASM
OTLP receiver start listening on `127.0.0.1:4318`.

Terminal 3 — send a trace:

```console
$ curl -X POST http://127.0.0.1:4318/v1/traces -H 'Content-Type: application/json' -d '{
  "resourceSpans": [{
    "resource": {"attributes": [{"key": "service.name", "value": {"stringValue": "opamp-demo"}}]},
    "scopeSpans": [{
      "scope": {"name": "demo"},
      "spans": [{
        "traceId": "5b8efff798038103d269b633813fc60c",
        "spanId": "eee19b7ec3c1b174",
        "name": "demo-span",
        "kind": 1,
        "startTimeUnixNano": "1784965200000000000",
        "endTimeUnixNano": "1784965201000000000"
      }]
    }]
  }]
}'
```

In terminal 2, the WASM stdout exporter prints the span with the attribute
added by the WASM processor, using the values from the OpAMP-distributed
config:

```
New attribute added: opamp.managed=hello-from-opamp
{"resourceSpans":[... {"key":"opamp.managed","value":{"stringValue":"hello-from-opamp"}} ...]}
```

## Updating the config remotely

While everything is running, edit `remote-config.yaml` — e.g. change
`attribute_value` to `"updated-by-opamp-push"`. The OpAMP server detects the
change and pushes the new config; the supervisor restarts the collector with
it. Send the trace again and the new value appears:

```
New attribute added: opamp.managed=updated-by-opamp-push
```

The server logs the agent's acknowledgement:

```
opamp-server: ... agent <uid> remote config status: RemoteConfigStatuses_APPLIED (hash=...)
```

Because the remote config controls `path` and `plugin_config` of each WASM
component, the same mechanism can swap entire WASM plugins at runtime, not
just tweak their settings.

## Combining with OCI plugin sources

Plugin `path` values in the remote config can also be `oci://` (or `http://`)
references, which turns this setup into fully remote plugin distribution: the
OpAMP server pushes a config that references an image, and the collector pulls
the WASM module from the registry — nothing needs to exist on the host
beforehand.

Push the demo plugins to a registry (see "Loading plugins from an OCI
registry" in the top-level README) and use them in `remote-config.yaml`:

```yaml
receivers:
  wasm/otlp:
    path: "oci://127.0.0.1:5001/otelwasm/otlpreceiver:e2e"
    plugin_config:
      protocols:
        http:
          endpoint: 127.0.0.1:4318

processors:
  wasm/addattr:
    path: "oci://127.0.0.1:5001/otelwasm/addattr:e2e"
    plugin_config:
      attribute_name: distribution
      attribute_value: opamp-config-oci-plugin

exporters:
  wasm/stdout:
    path: "oci://127.0.0.1:5001/otelwasm/stdoutexporter:e2e"

service:
  pipelines:
    traces:
      receivers: [wasm/otlp]
      processors: [wasm/addattr]
      exporters: [wasm/stdout]
```

To swap a plugin remotely, push a different module (e.g.
`examples/processor/attributesprocessor`) under a new reference, point the
processor's `path` at it, and adjust `plugin_config` accordingly. The OpAMP
server pushes the new config and the collector restarts, pulling the new
plugin from the registry.

Note for macOS: port 5000 is often taken by the AirPlay Receiver, which
answers `403 Forbidden` on `localhost:5000`. Run the local registry on
another port (e.g. 5001) or use `127.0.0.1` explicitly.
