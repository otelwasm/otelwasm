Output sample1.mp4
Set Shell bash
Set FontSize 36
Set Width 1280
Set Height 720
Set Framerate 60
Set Padding 10
Set TypingSpeed 0.2
Set WaitTimeout 30s

Type "# Create a new directory for the demo"
Enter
Type "cd $(mktemp -d)"
Enter
Wait
Sleep 3s

Enter
Type "# Let's build WASM modules for otel components!"
Enter
Enter
Wait
Sleep 1s

Type "# Build otlpreceiver"
Enter
Sleep 1s
Type "factorybuilder go.opentelemetry.io/collector/receiver/otlpreceiver"
Enter
Enter
Wait
Sleep 5s

Type "# Add receiver to the otel-collector config"
Enter
Type "vim otel-config.yaml"
Enter
Sleep 1s
Type "i"
Sleep 1s

Type `receivers:`
Enter
Type `  wasm/otlpreceiver:`
Enter
Type `    path: "./otlpreceiver.wasm"`
Enter
Type `    plugin_config:`
Enter
Type `      protocols:`
Enter
Type `        http:`
Enter
Type `          endpoint: "localhost:4318"`
Enter
Enter
Sleep 5s

Ctrl+C
Type ":wq"
Enter
Wait
Sleep 3s

Type "clear"
Enter
Sleep 1s

Type "# Build attributesprocessor"
Enter
Sleep 1s
Type "factorybuilder github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
Enter
Enter
Wait
Sleep 5s

Type "# Add processor to the otel-collector config"
Enter
Type "vim otel-config.yaml"
Enter
Sleep 1s
Shift+G
Type "i"
Sleep 1s

Type `processors:`
Enter
Type `  wasm/attributesprocessor:`
Enter
Type `    path: "./attributesprocessor.wasm"`
Enter
Type `    plugin_config:`
Enter
Type `      actions:`
Enter
Type `        - key: "event"`
Enter
Type `          value: "cnds2025"`
Enter
Type `          action: "insert"`
Enter
Enter
Sleep 5s

Ctrl+C
Type ":wq"
Enter
Wait
Sleep 3s

Type "clear"
Enter
Sleep 1s

Type "# Build stdoutexporter"
Enter
Sleep 1s
Type "git clone https://github.com/otelwasm/otelwasm.git"
Enter
Wait
Sleep 1s
Type "pushd otelwasm/examples/exporter/stdout"
Enter
Sleep 3s
Type "wasibuilder go build -buildmode=c-shared -o stdoutexporter.wasm ."
Enter
Wait
Sleep 5s
Type "popd"
Enter
Wait
Sleep 3s
Type "cp otelwasm/examples/exporter/stdout/stdoutexporter.wasm ."
Enter
Wait
Sleep 5s

Type "# Add exporter to the otel-collector config"
Enter
Type "vim otel-config.yaml"
Enter
Sleep 1s
Shift+G
Type "i"
Sleep 1s

Type `exporters:`
Enter
Type `  wasm/stdoutexporter:`
Enter
Type `    path: ./stdoutexporter.wasm`
Enter
Type `    plugin_config:`
Enter
Type `      sending_queue:`
Enter
Type `        enabled: false`
Enter
Sleep 5s

Enter
Type `service:`
Enter
Type `  pipelines:`
Enter
Type `    logs:`
Enter
Type `      receivers: [wasm/otlpreceiver]`
Enter
Type `      processors: [wasm/attributesprocessor]`
Enter
Type `      exporters: [wasm/stdoutexporter]`
Enter
Sleep 5s

Ctrl+C
Type ":wq"
Enter
Wait
Sleep 3s

Type "clear"
Enter
Sleep 1s

Type "bat otel-config.yaml"
Enter
Down
Sleep 1s
Down
Sleep 1s
Down
Sleep 1s
Down
Sleep 1s
Down
Sleep 1s
Down
Sleep 1s
Type "q"
Wait
Sleep 5s

Type "tmux"
Enter
Sleep 1s

Type `otelwasmcol --config otel-config.yaml | grep --color -e cnds2025 -e 'Hello OTLP log!'`
Enter
Sleep 5s

Ctrl+S
Type `"`
Sleep 5s

Type `curl https://gist.githubusercontent.com/`
Type@0ms `tsuzu/0de1f1203f491719ff78fc0d22e881b3/raw/94192ac4544c4d2ec6a5d4f95908723af9ac4079/otlp-log.json`
Type ` > log.json`
Enter
Sleep 3s

Type `cat log.json | jq -10 --color "Hello"`
Enter
Sleep 5s

Type `curl -v -X POST \`
Enter
Sleep 1s
Type `  -H "Content-Type: application/json" \`
Enter
Sleep 1s
Type `  --data-binary @log.json \`
Enter
Sleep 1s
Type `  http://localhost:4318/v1/logs`
Enter
Sleep 10s

Type "exit"
Enter
Sleep 1s

Ctrl+C

Type "exit"
Enter
Sleep 1s
