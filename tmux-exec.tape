Type "tmux"
Enter
Sleep 1s

Type `otelcol-contrib --config otel-config.yaml | grep --color -e cnds2025 -e 'Hello OTLP log!'`
Enter
Sleep 5s

Ctrl+S
Type `"`
Sleep 5s

Type `cat ../otlp-log.json | grep -10 --color 'Hello OTLP log!'`
Enter
Sleep 5s

Type `curl -v -X POST \`
Enter
Sleep 1s
Type `  -H "Content-Type: application/json" \`
Enter
Sleep 1s
Type `  --data-binary @../otlp-log.json \`
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
