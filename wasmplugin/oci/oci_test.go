package oci

import (
	"bytes"
	"encoding/json"
	"testing"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
)

var fakeWasm = []byte("\x00asm\x01\x00\x00\x00 fake module")

func TestPushPullWasmRoundTrip(t *testing.T) {
	metadata := []byte(`{"name":"nop"}`)
	for _, format := range []Format{FormatWasm, ""} {
		t.Run(string(format), func(t *testing.T) {
			ctx := t.Context()
			store := memory.New()

			if _, err := Push(ctx, store, "v1", fakeWasm, metadata, format); err != nil {
				t.Fatalf("push failed: %v", err)
			}
			got, err := PullTarget(ctx, store, "v1")
			if err != nil {
				t.Fatalf("pull failed: %v", err)
			}
			if !bytes.Equal(got, fakeWasm) {
				t.Fatal("pulled wasm differs from pushed wasm")
			}

			desc, err := oras.Resolve(ctx, store, "v1", oras.DefaultResolveOptions)
			if err != nil {
				t.Fatalf("resolve failed: %v", err)
			}
			manifestBytes, err := content.FetchAll(ctx, store, desc)
			if err != nil {
				t.Fatalf("fetch manifest failed: %v", err)
			}
			var manifest v1.Manifest
			if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
				t.Fatalf("parse manifest failed: %v", err)
			}
			if manifest.Config.MediaType != MediaTypeWasmConfig {
				t.Fatalf("config media type = %s, want %s", manifest.Config.MediaType, MediaTypeWasmConfig)
			}
			if got := manifest.Annotations[AnnotationMetadata]; got != string(metadata) {
				t.Fatalf("metadata annotation = %q, want %q", got, metadata)
			}
		})
	}
}

func TestPullRejectsMultiLayerWasmArtifact(t *testing.T) {
	ctx := t.Context()
	store := memory.New()

	layer1, err := pushBlob(ctx, store, MediaTypeWasmLayer, fakeWasm)
	if err != nil {
		t.Fatalf("push layer failed: %v", err)
	}
	layer2, err := pushBlob(ctx, store, MediaTypeWasmLayer, []byte("second layer"))
	if err != nil {
		t.Fatalf("push layer failed: %v", err)
	}
	configDesc, err := pushBlob(ctx, store, MediaTypeWasmConfig, []byte("{}"))
	if err != nil {
		t.Fatalf("push config failed: %v", err)
	}
	manifest, err := json.Marshal(v1.Manifest{
		MediaType: v1.MediaTypeImageManifest,
		Config:    configDesc,
		Layers:    []v1.Descriptor{layer1, layer2},
	})
	if err != nil {
		t.Fatalf("marshal manifest failed: %v", err)
	}
	manifestDesc, err := pushBlob(ctx, store, v1.MediaTypeImageManifest, manifest)
	if err != nil {
		t.Fatalf("push manifest failed: %v", err)
	}
	if err := store.Tag(ctx, manifestDesc, "v1"); err != nil {
		t.Fatalf("tag failed: %v", err)
	}

	if _, err := PullTarget(ctx, store, "v1"); err == nil {
		t.Fatal("expected error for multi-layer wasm artifact")
	}
}

func TestPullLegacyArtifact(t *testing.T) {
	ctx := t.Context()
	store := memory.New()

	// Build a legacy otelwasm artifact (the removed FormatArtifact layout).
	wasmDesc, err := pushBlob(ctx, store, MediaTypeContentLayer, fakeWasm)
	if err != nil {
		t.Fatalf("push wasm layer failed: %v", err)
	}
	metadataDesc, err := pushBlob(ctx, store, MediaTypeMetadata, []byte(`{"name":"nop"}`))
	if err != nil {
		t.Fatalf("push metadata failed: %v", err)
	}
	manifestDesc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, "application/vnd.otelwasm.plugin.v1+json", oras.PackManifestOptions{
		ConfigDescriptor: &metadataDesc,
		Layers:           []v1.Descriptor{wasmDesc},
	})
	if err != nil {
		t.Fatalf("pack manifest failed: %v", err)
	}
	if err := store.Tag(ctx, manifestDesc, "v1"); err != nil {
		t.Fatalf("tag failed: %v", err)
	}

	got, err := PullTarget(ctx, store, "v1")
	if err != nil {
		t.Fatalf("pull failed: %v", err)
	}
	if !bytes.Equal(got, fakeWasm) {
		t.Fatal("pulled wasm differs from pushed wasm")
	}
}

func TestPushPullCompatRoundTrip(t *testing.T) {
	ctx := t.Context()
	store := memory.New()

	if _, err := Push(ctx, store, "v1", fakeWasm, nil, FormatCompat); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	got, err := PullTarget(ctx, store, "v1")
	if err != nil {
		t.Fatalf("pull failed: %v", err)
	}
	if !bytes.Equal(got, fakeWasm) {
		t.Fatal("pulled wasm differs from pushed wasm")
	}
}

func TestPushRejectsUnknownFormat(t *testing.T) {
	for _, format := range []Format{"artifact", "bogus"} {
		if _, err := Push(t.Context(), memory.New(), "v1", fakeWasm, nil, format); err == nil {
			t.Fatalf("expected error for format %q", format)
		}
	}
}
