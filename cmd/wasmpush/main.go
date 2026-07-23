// Command wasmpush publishes an otelwasm plugin module to an OCI registry.
//
// Usage:
//
//	wasmpush [-metadata metadata.json] [-format artifact|compat] main.wasm ghcr.io/otelwasm/nopprocessor:latest
//
// Registry credentials are read from the local docker config (docker login).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/otelwasm/otelwasm/wasmplugin/oci"
)

var (
	metadataPath string
	format       string
)

func main() {
	flag.StringVar(&metadataPath, "metadata", "", "path to plugin metadata JSON file (optional)")
	flag.StringVar(&format, "format", string(oci.FormatArtifact), "image format: artifact (OCI artifact) or compat (standard container image for registries without OCI artifact support)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <wasm-file> <reference>\n\nExample:\n  %s main.wasm ghcr.io/otelwasm/nopprocessor:latest\n\nFlags:\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(context.Background(), flag.Arg(0), flag.Arg(1)); err != nil {
		fmt.Fprintf(os.Stderr, "wasmpush: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, wasmPath, ref string) error {
	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		return err
	}

	var metadata []byte
	if metadataPath != "" {
		if metadata, err = os.ReadFile(metadataPath); err != nil {
			return err
		}
	}

	repo, err := oci.NewRepository(ref)
	if err != nil {
		return err
	}
	tag := repo.Reference.Reference
	if tag == "" {
		return fmt.Errorf("reference %s has no tag", ref)
	}

	desc, err := oci.Push(ctx, repo, tag, wasm, metadata, oci.Format(format))
	if err != nil {
		return err
	}
	fmt.Printf("pushed %s\ndigest: %s\n", ref, desc.Digest)
	return nil
}
