Output sample-nonwasm.mp4
Set Shell bash
Set FontSize 36
Set Width 1280
Set Height 720
Set Framerate 10
Set Padding 10
Set TypingSpeed 0.15
Set WaitTimeout 30s


Type "# Add otel-collector config"
Enter
Sleep 3s
Type "vim otel-config.yaml"
Enter
Sleep 1s
Type "i"
Sleep 1s

Type `receivers:`
Enter
Type `  otlp:`
Enter
Type `    protocols:`
Enter
Type `      http:`
Enter
Type `        endpoint: "localhost:4318"`
Enter
Enter
Sleep 5s

Type `processors:`
Enter
Type `  attributes:`
Enter
Type `    actions:`
Enter
Type `      - key: "event"`
Enter
Type `        value: "cnds2025"`
Enter
Type `        action: "insert"`
Enter
Enter
Sleep 5s

Type `exporters:`
Enter
Type `  stdout:`
Enter
Sleep 5s

Enter
Type `service:`
Enter
Type `  pipelines:`
Enter
Type `    logs:`
Enter
Type `      receivers: [otlp]`
Enter
Type `      processors: [attributes]`
Enter
Type `      exporters: [stdout]`
Enter
Sleep 5s

Ctrl+C
Type ":wq"
Enter
Wait
Sleep 3s

Type "clear"
Enter
Sleep 3s

Source ../tmux-exec.tape
