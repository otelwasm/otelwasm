# Why we use Interpreter runtime instead of the Compiler runtime?

We encountered the problem where the compiler runtime caused panic in some cases. This could be a bug in the compiler runtime or a bug in the code we wrote, but we haven't concluded yet. The fact that the interpreter runtime works fine in all cases suggests that the compiler runtime may not be as stable as we would like it to be.

For more details, see https://github.com/otelwasm/otelwasm/pull/27.

## ABI v1 push model rationale in `wasmplugin`

`wasmplugin` now treats ABI v1 as a strict boundary for push-model components:

- ABI v1 modules are validated by the `abi_version_v1` export marker.
- Required guest exports (for example `otelwasm_consume_traces`) are validated at module initialization.
- Telemetry flows via host push:
  - host marshals data
  - host calls guest `alloc`
  - host writes payload into guest memory
  - host calls guest `consume_*`
- Receiver ABI v1 entrypoints are `otelwasm_start_traces_receiver`, `otelwasm_start_metrics_receiver`, and `otelwasm_start_logs_receiver`.

For error reporting, non-zero `consume_*` status codes are surfaced with status strings (`ERROR`, etc.) and include guest-provided reason text set through `set_status_reason` when available.
