Output sample1.mp4
Set Shell zsh
Set FontSize 12
Set Width 600
Set Height 450
Set Framerate 60
Set Padding 10

Type ": Create a new directory for the demo"
Enter
Type "rm -r demo2"
Enter
Type "mkdir demo2; cd demo2"
Enter
Wait
Sleep 3s

Enter
Type ': Create a new Go module'
Enter
Type "go mod init cnds2025demo"
Enter
Wait

Enter
Type ": Install dependencies"
Enter
Type "go get github.com/otelwasm/otelwasm/guest"
Enter
Wait

Enter
Type "go get go.opentelemetry.io/collector/pdata/plog"
Enter
Wait
Sleep 5s

Enter
Type ": Create a new Go file"
Enter
Sleep 2s

# Open the Go file in a text editor
Type "vim main.go"
Enter
Sleep 1s
Type ":syntax on"
Enter
Sleep 1s
Type "i"
Sleep 1s
# Write the Go code
Type `package main`
Enter
Type ``
Enter
Type `import (`
Enter
Type `  "go.opentelemetry.io/collector/pdata/plog"`
Enter
Type ``
Enter
Type `  // Import guest API`
Enter
Type `  "github.com/otelwasm/otelwasm/guest/api"`
Enter
Type `  "github.com/otelwasm/otelwasm/guest/plugin"`
Enter
Type `)`
Enter
Type ``
Enter
Type `type processor struct {`
Enter
Type `}`
Enter
Type ``
Enter
Type `var _ api.LogsProcessor = (*processor)(nil)`
Enter
Type ``
Enter

Type `func (p *processor) ProcessLogs(logs plog.Logs) (plog.Logs, *api.Status) {`
Enter
Type `  // Add 'event' attribute to each log record`
Enter
Sleep 3s
Enter
Type `  for _, resourceLogs := range logs.ResourceLogs().All() {`
Enter
Type `    for _, scopeLog := range resourceLogs.ScopeLogs().All() {`
Enter
Type `      for _, logRecord := range scopeLog.LogRecords().All() {`
Enter
Type `        logRecord.Attributes().PutStr("event", "cnds2025")`
Enter
Type `      }`
Enter
Type `    }`
Enter
Type `  }`
Enter
Type ``
Enter
Type `  return logs, nil`
Enter
Sleep 3s
Type `}`
Enter


Type ``
Enter
Type `func init() {`
Enter
Type `  // Register the processor as logs processor`
Enter
Type `  plugin.Set(&processor{})`
Enter
Sleep 3s
Type `}`
Enter
Type ``
Enter
Type `func main() {`
Enter
Type `}`
Enter
Sleep 10s

Ctrl+C
Type ":wq"
Enter
Wait

Sleep 1s
Type "clear"
Enter

Type ": Build the Go code into a WebAssembly module with wasibuilder"
Enter
# Install wasibuilder
Type "go install github.com/otelwasm/wasibuilder@latest"
Enter
Wait

# Build the Go code into a WebAssembly module
Type "wasibuilder go build -o cnds2025demo.wasm"
Enter
Wait

Type "ls -l"
Enter
Wait
Sleep 3s

Type "clear"
Enter
Sleep 1s

Type ": Prepare the OpenTelemetry Collector configuration"
Type "cp ../otel-config-demo.yaml ."
Enter
Wait
Sleep 3s

Type "bat otel-config-demo.yaml"
Enter
Wait
Sleep 5s

Type "otelwasm"
