Output sample1.mp4
Set Shell bash
Set FontSize 36
Set Width 1280
Set Height 720
Set Framerate 10
Set Padding 10
Set TypingSpeed 0.15
Set WaitTimeout 30s

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
Source ../edit-receiver.tape

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
Source ../edit-processor.tape

Type "clear"
Enter
Sleep 1s

Type "# Build stdoutexporter"
Enter
Sleep 1s
Type "factorybuilder github.com/tsuzu/stdoutexporter"
Enter
Enter
Wait
Sleep 5s

Type "# Add exporter to the otel-collector config"
Enter
Source ../edit-exporter.tape

Type "clear"
Enter
Sleep 1s

Source ../tmux-exec-wasm.tape
