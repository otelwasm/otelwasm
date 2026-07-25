# Why we use Interpreter runtime instead of the Compiler runtime?

We originally encountered a compiler-runtime panic in some cases. That issue was later resolved by [#83](https://github.com/otelwasm/otelwasm/pull/83) and its follow-up [#84](https://github.com/otelwasm/otelwasm/pull/84). We still prefer interpreter mode because it is more stable than compilation mode.

For more details, see https://github.com/otelwasm/otelwasm/pull/27.
