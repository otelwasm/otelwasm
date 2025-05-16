package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

var (
	output        string
	componentType ComponentType
	workDir       string
	remain        bool
)

func init() {
	flag.StringVar(&output, "o", "", "output file (default: {package}.wasm)")
	flag.StringVar((*string)(&componentType), "type", "", "component type: receiver, processor, exporter (default: detect from package)")
	flag.StringVar(&workDir, "workdir", "", "working directory (default: ./{package})")
	flag.BoolVar(&remain, "remain", false, "keep the working directory after build")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s {package}\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
}

func detectComponentType(packagePath string) ComponentType {
	switch {
	case strings.Contains(packagePath, "receiver"):
		return Receiver
	case strings.Contains(packagePath, "processor"):
		return Processor
	case strings.Contains(packagePath, "exporter"):
		return Exporter
	default:
		return ""
	}
}

func main() {
	if flag.NArg() == 0 {
		flag.Usage()
		return
	}
	packagePath := flag.Arg(0)
	split := strings.Split(packagePath, "/")
	packageName := split[len(split)-1]

	if output == "" {
		output = packageName + ".wasm"
	}

	if componentType == "" {
		componentType = detectComponentType(packagePath)
	}
	if componentType == "" {
	}
	switch componentType {
	case Receiver, Processor, Exporter:
		// OK
	case "":
		slog.Error("Could not detect component type from package path", "packagePath", packagePath)
		slog.Info("Please specify the component type using -type flag")
		os.Exit(1)
	default:
		slog.Error("Invalid component type", "componentType", componentType)
		slog.Info("Valid component types are: receiver, processor, exporter")
		os.Exit(1)
	}

	if workDir == "" {
		workDir = packageName
	}

	builder := &Builder{
		WorkDir:       workDir,
		ComponentType: componentType,
		Package:       packagePath,
		PackageName:   packageName,
		Output:        output,
	}

	exitCode := 0
	defer func() {
		recovered := recover()

		if remain {
			slog.Info("Working directory will be kept", "workDir", workDir)

			os.Exit(exitCode)
		}

		err := builder.Clean()
		if err != nil {
			slog.Warn("Failed to clean up", "error", err)
		}

		if recovered != nil {
			os.Exit(exitCode)
		}
	}()

	err := builder.Prepare()
	if err != nil {
		slog.Error("Failed to prepare build", "error", err)
		panic(err)
	}
	err = builder.Build()
	if err != nil {
		slog.Error("Failed to build package", "error", err)
		panic(err)
	}

	slog.Info("Build completed successfully", "output", output)
}
