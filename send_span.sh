#!/bin/bash

set -euo pipefail

# configure otel-cli to talk the the local server spawned above

# GRPC
#export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
# HTTP
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# run a program inside a span
#otel-cli exec --service my-service --name "curl google" curl https://google.com
otel-cli exec --service my-service --name "curl google" curl https://google.com
