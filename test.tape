Output test.mp4
Set Shell bash
Set FontSize 36
Set Width 1280
Set Height 720
Set Framerate 10
Set Padding 10
Set TypingSpeed 0.2
Set WaitTimeout 30s

Type "vim otel-config.yaml"
Enter
Sleep 1s
Type "i"
Sleep 1s

Down
Right
Right
Type `wasm/`
Escape
Type "[F"
Enter
Type `    path: "./otlpreceiver.wasm"`
Enter
Type `    plugin_config:`
Escape
Type "[H"
Down
Type `  `
Down
Type `  `
Down
Type `  `
Down

Ctrl+C
Type `gg`
Type "i"
Sleep 1s

Down 9
Right
Right
Type `wasm/`
Escape
Type "[F"
Enter
Type `    path: "./attributesprocessor.wasm"`
Enter
Type `    plugin_config:`
Escape
Type "[H"
Down
Type `  `
Down
Type `  `
Down
Type `  `
Down
Type `  `
Down

Ctrl+C
Type `gg`
Type "i"
Sleep 1s

Down 18
Right
Right
Type `wasm/`
Escape
Type "[F"
Enter
Type `    path: "./stdoutexporter.wasm"`
Escape
Type "[H"
Down

Sleep 5s

Down 4
Right 18
Type `wasm/`
Left 4
Down
Type `wasm/`
Left 6
Down
Type `wasm/`

Sleep 5s

Ctrl+C
Type ":q!"
Enter
Wait
Sleep 3s
