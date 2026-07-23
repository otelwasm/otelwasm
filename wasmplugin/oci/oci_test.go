package oci

import (
	"bytes"
	"testing"

	"oras.land/oras-go/v2/content/memory"
)

var fakeWasm = []byte("\x00asm\x01\x00\x00\x00 fake module")

func TestPushPullArtifactRoundTrip(t *testing.T) {
	ctx := t.Context()
	store := memory.New()

	if _, err := Push(ctx, store, "v1", fakeWasm, []byte(`{"name":"nop"}`), FormatArtifact); err != nil {
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
	_, err := Push(t.Context(), memory.New(), "v1", fakeWasm, nil, Format("bogus"))
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}
