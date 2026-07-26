// Package oci implements pushing and pulling otelwasm plugin modules
// as OCI images.
//
// Two image formats are supported for push:
//   - FormatWasm: the CNCF Wasm OCI Artifact layout, see
//     https://tag-runtime.cncf.io/wgs/wasm/deliverables/wasm-oci-artifact/.
//   - FormatCompat: a standard container image whose single tar.gz layer
//     contains plugin.wasm, for registries without OCI artifact support.
//     See https://github.com/solo-io/wasm/blob/master/spec/spec-compat.md.
//
// Pull transparently handles both formats, plus legacy artifacts pushed
// with the old otelwasm media types.
package oci

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	godigest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	// MediaTypeWasmConfig is the config media type of the CNCF Wasm OCI
	// Artifact layout; it identifies the artifact as Wasm.
	MediaTypeWasmConfig = "application/vnd.wasm.config.v0+json"

	// MediaTypeWasmLayer is the layer media type holding the raw wasm bytes
	// in the CNCF Wasm OCI Artifact layout.
	MediaTypeWasmLayer = "application/wasm"

	// AnnotationMetadata is the manifest annotation storing the otelwasm
	// plugin metadata JSON in the wasm format.
	AnnotationMetadata = "dev.otelwasm.plugin.metadata"

	// MediaTypeMetadata is the media type of the otelwasm plugin metadata
	// in the legacy artifact format (pull only).
	MediaTypeMetadata = "application/vnd.otelwasm.plugin.metadata.v1+json"

	// MediaTypeContentLayer is the media type of the compiled plugin module
	// data in the legacy artifact format (pull only).
	MediaTypeContentLayer = "application/vnd.otelwasm.plugin.content.layer.v1+wasm"

	// compatWasmFilename is the name of the wasm module inside the tar.gz
	// layer of a compat format image.
	compatWasmFilename = "plugin.wasm"

	mediaTypeDockerManifest = "application/vnd.docker.distribution.manifest.v2+json"
	mediaTypeDockerConfig   = "application/vnd.docker.container.image.v1+json"
	mediaTypeDockerLayer    = "application/vnd.docker.image.rootfs.diff.tar.gzip"
)

// Format is the OCI image format used to store the plugin.
type Format string

const (
	// FormatWasm stores the plugin in the CNCF Wasm OCI Artifact layout.
	FormatWasm Format = "wasm"

	// FormatCompat stores the plugin as a standard container image whose
	// single tar.gz layer contains plugin.wasm. Use this for registries that
	// don't support OCI artifacts.
	FormatCompat Format = "compat"
)

// Pull fetches the wasm module stored in the OCI image referenced by ref
// (e.g. "ghcr.io/otelwasm/nopprocessor:latest"). The wasm, compat, and
// legacy otelwasm artifact formats are all handled.
func Pull(ctx context.Context, ref string) ([]byte, error) {
	repo, err := NewRepository(ref)
	if err != nil {
		return nil, err
	}
	return PullTarget(ctx, repo, repo.Reference.Reference)
}

// PullTarget fetches the wasm module tagged with tag from any ORAS target.
func PullTarget(ctx context.Context, target oras.ReadOnlyTarget, tag string) ([]byte, error) {
	desc, err := oras.Resolve(ctx, target, tag, oras.DefaultResolveOptions)
	if err != nil {
		return nil, fmt.Errorf("oci: failed to resolve %s: %w", tag, err)
	}

	manifestBytes, err := content.FetchAll(ctx, target, desc)
	if err != nil {
		return nil, fmt.Errorf("oci: failed to fetch manifest: %w", err)
	}
	var manifest v1.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("oci: failed to parse manifest: %w", err)
	}

	if manifest.Config.MediaType == MediaTypeWasmConfig {
		if len(manifest.Layers) != 1 {
			return nil, fmt.Errorf("oci: wasm artifact must have exactly one layer, got %d", len(manifest.Layers))
		}
		layer := manifest.Layers[0]
		if layer.MediaType != MediaTypeWasmLayer {
			return nil, fmt.Errorf("oci: wasm artifact layer has unexpected media type %s (expected %s)", layer.MediaType, MediaTypeWasmLayer)
		}
		wasm, err := content.FetchAll(ctx, target, layer)
		if err != nil {
			return nil, fmt.Errorf("oci: failed to fetch wasm layer: %w", err)
		}
		return wasm, nil
	}

	for _, layer := range manifest.Layers {
		switch layer.MediaType {
		case MediaTypeWasmLayer, MediaTypeContentLayer:
			wasm, err := content.FetchAll(ctx, target, layer)
			if err != nil {
				return nil, fmt.Errorf("oci: failed to fetch wasm layer: %w", err)
			}
			return wasm, nil
		case mediaTypeDockerLayer, v1.MediaTypeImageLayerGzip, v1.MediaTypeImageLayer:
			blob, err := content.FetchAll(ctx, target, layer)
			if err != nil {
				return nil, fmt.Errorf("oci: failed to fetch compat layer: %w", err)
			}
			return extractWasmFromTar(blob)
		}
	}
	return nil, fmt.Errorf("oci: no wasm layer found in manifest (expected media type %s, %s, or a compat tar.gz layer)", MediaTypeWasmLayer, MediaTypeContentLayer)
}

// Push packs the wasm module and metadata JSON in the given format and pushes
// it to the target with the given tag. Non-empty metadata is stored as a
// manifest annotation in the wasm format. Metadata is ignored by the compat
// format, which has no dedicated slot for it. It returns the manifest
// descriptor.
func Push(ctx context.Context, target oras.Target, tag string, wasm, metadata []byte, format Format) (v1.Descriptor, error) {
	store := memory.New()

	var manifestDesc v1.Descriptor
	var err error
	switch format {
	case FormatWasm, "":
		manifestDesc, err = packWasm(ctx, store, wasm, metadata)
	case FormatCompat:
		manifestDesc, err = packCompat(ctx, store, wasm)
	default:
		return v1.Descriptor{}, fmt.Errorf("oci: unsupported format: %s", format)
	}
	if err != nil {
		return v1.Descriptor{}, err
	}

	if err := store.Tag(ctx, manifestDesc, tag); err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to tag manifest: %w", err)
	}
	desc, err := oras.Copy(ctx, store, tag, target, tag, oras.DefaultCopyOptions)
	if err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to push: %w", err)
	}
	return desc, nil
}

// packWasm stores the wasm module in the CNCF Wasm OCI Artifact layout: a
// single application/wasm layer with a wasm config blob. Non-empty metadata
// is stored as a manifest annotation.
func packWasm(ctx context.Context, store *memory.Store, wasm, metadata []byte) (v1.Descriptor, error) {
	layerDesc, err := pushBlob(ctx, store, MediaTypeWasmLayer, wasm)
	if err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to store wasm layer: %w", err)
	}

	config, err := json.Marshal(struct {
		Created      string            `json:"created"`
		Architecture string            `json:"architecture"`
		OS           string            `json:"os"`
		LayerDigests []godigest.Digest `json:"layerDigests"`
	}{
		Created:      time.Now().UTC().Format(time.RFC3339),
		Architecture: "wasm",
		OS:           "wasip1",
		LayerDigests: []godigest.Digest{layerDesc.Digest},
	})
	if err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to marshal wasm config: %w", err)
	}
	configDesc, err := pushBlob(ctx, store, MediaTypeWasmConfig, config)
	if err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to store wasm config: %w", err)
	}

	var annotations map[string]string
	if len(metadata) > 0 {
		annotations = map[string]string{AnnotationMetadata: string(metadata)}
	}
	manifest, err := json.Marshal(v1.Manifest{
		Versioned:   specs.Versioned{SchemaVersion: 2},
		MediaType:   v1.MediaTypeImageManifest,
		Config:      configDesc,
		Layers:      []v1.Descriptor{layerDesc},
		Annotations: annotations,
	})
	if err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to marshal manifest: %w", err)
	}
	return pushBlob(ctx, store, v1.MediaTypeImageManifest, manifest)
}

// packCompat stores a docker-compatible image manifest whose single tar.gz
// layer contains plugin.wasm.
func packCompat(ctx context.Context, store *memory.Store, wasm []byte) (v1.Descriptor, error) {
	layer, diffID, err := buildWasmTarGz(wasm)
	if err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to build compat layer: %w", err)
	}
	layerDesc, err := pushBlob(ctx, store, mediaTypeDockerLayer, layer)
	if err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to store compat layer: %w", err)
	}

	config, err := json.Marshal(v1.Image{
		Config: v1.ImageConfig{},
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: []godigest.Digest{diffID},
		},
	})
	if err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to marshal image config: %w", err)
	}
	configDesc, err := pushBlob(ctx, store, mediaTypeDockerConfig, config)
	if err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to store image config: %w", err)
	}

	manifest, err := json.Marshal(v1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		MediaType: mediaTypeDockerManifest,
		Config:    configDesc,
		Layers:    []v1.Descriptor{layerDesc},
	})
	if err != nil {
		return v1.Descriptor{}, fmt.Errorf("oci: failed to marshal manifest: %w", err)
	}
	return pushBlob(ctx, store, mediaTypeDockerManifest, manifest)
}

// buildWasmTarGz wraps the wasm module in a tar.gz archive as plugin.wasm and
// returns the archive with the diff ID (digest of the uncompressed tar).
func buildWasmTarGz(wasm []byte) ([]byte, godigest.Digest, error) {
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	if err := tw.WriteHeader(&tar.Header{
		Name: compatWasmFilename,
		Mode: 0o644,
		Size: int64(len(wasm)),
	}); err != nil {
		return nil, "", err
	}
	if _, err := tw.Write(wasm); err != nil {
		return nil, "", err
	}
	if err := tw.Close(); err != nil {
		return nil, "", err
	}
	diffID := content.NewDescriptorFromBytes("", tarBuf.Bytes()).Digest

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	if _, err := gw.Write(tarBuf.Bytes()); err != nil {
		return nil, "", err
	}
	if err := gw.Close(); err != nil {
		return nil, "", err
	}
	return gzBuf.Bytes(), diffID, nil
}

// extractWasmFromTar extracts plugin.wasm from a (possibly gzipped) tar layer.
func extractWasmFromTar(blob []byte) ([]byte, error) {
	var r io.Reader = bytes.NewReader(blob)
	if gz, err := gzip.NewReader(bytes.NewReader(blob)); err == nil {
		defer gz.Close()
		r = gz
	}

	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("oci: failed to read compat layer tar: %w", err)
		}
		if strings.TrimPrefix(hdr.Name, "./") == compatWasmFilename {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("oci: %s not found in compat layer", compatWasmFilename)
}

// NewRepository returns a remote repository client for the given reference,
// authenticating with credentials from the local docker config. Plain HTTP is
// used for localhost registries.
func NewRepository(ref string) (*remote.Repository, error) {
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, fmt.Errorf("oci: invalid reference %s: %w", ref, err)
	}
	repo.PlainHTTP = isLocalhost(repo.Reference.Registry)

	credStore, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return nil, fmt.Errorf("oci: failed to load docker credentials: %w", err)
	}
	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore),
	}
	return repo, nil
}

func isLocalhost(host string) bool {
	host, _, _ = strings.Cut(host, ":")
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func pushBlob(ctx context.Context, store *memory.Store, mediaType string, blob []byte) (v1.Descriptor, error) {
	desc := content.NewDescriptorFromBytes(mediaType, blob)
	if err := store.Push(ctx, desc, bytes.NewReader(blob)); err != nil && !errors.Is(err, errdef.ErrAlreadyExists) {
		return v1.Descriptor{}, err
	}
	return desc, nil
}
