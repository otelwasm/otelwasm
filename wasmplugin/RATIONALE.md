# Why we use Interpreter runtime instead of the Compiler runtime?

We encountered the problem where the compiler runtime caused panic in some cases. This could be a bug in the compiler runtime or a bug in the code we wrote, but we haven't concluded yet. The fact that the interpreter runtime works fine in all cases suggests that the compiler runtime may not be as stable as we would like it to be.

For more details, see https://github.com/otelwasm/otelwasm/pull/27.