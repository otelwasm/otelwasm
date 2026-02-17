package wasmplugin

import (
	"context"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func TestDetectABIVersion(t *testing.T) {
	tests := []struct {
		name    string
		exports []string
		want    ABIVersion
		nilMod  bool
	}{
		{
			name:    "detects v1 marker with otelwasm ABI export name",
			exports: []string{"otelwasm_abi_version_0_1_0"},
			want:    ABIV1,
		},
		{
			name:    "returns unknown for legacy v1 marker export name",
			exports: []string{"abi_version_v1"},
			want:    ABIUnknown,
		},
		{
			name:    "returns unknown when marker is absent",
			exports: []string{"some_other_export"},
			want:    ABIUnknown,
		},
		{
			name:   "returns unknown for nil module",
			nilMod: true,
			want:   ABIUnknown,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var got ABIVersion
			if tt.nilMod {
				got = detectABIVersion(nil)
			} else {
				mod := newTestModule(t, tt.exports)
				got = detectABIVersion(mod)
			}

			if got != tt.want {
				t.Fatalf("detectABIVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newTestModule(t *testing.T, exports []string) api.Module {
	t.Helper()

	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	t.Cleanup(func() {
		if err := r.Close(ctx); err != nil {
			t.Fatalf("failed to close runtime: %v", err)
		}
	})

	mod, err := r.Instantiate(ctx, newTestWasmModuleBytes(exports))
	if err != nil {
		t.Fatalf("failed to instantiate module: %v", err)
	}

	return mod
}

func newTestWasmModuleBytes(exports []string) []byte {
	module := []byte{
		0x00, 0x61, 0x73, 0x6d, // magic
		0x01, 0x00, 0x00, 0x00, // version
	}

	appendSection := func(sectionID byte, payload []byte) {
		module = append(module, sectionID)
		module = append(module, encodeULEB128(uint32(len(payload)))...)
		module = append(module, payload...)
	}

	// Type section: one `func ()`.
	appendSection(0x01, []byte{
		0x01,       // 1 type
		0x60, 0x00, // func, 0 params
		0x00, // 0 results
	})

	// Function section: one function using type index 0.
	appendSection(0x03, []byte{
		0x01, // 1 function
		0x00, // type index 0
	})

	if len(exports) > 0 {
		payload := append([]byte{}, encodeULEB128(uint32(len(exports)))...)
		for _, exportName := range exports {
			payload = append(payload, encodeULEB128(uint32(len(exportName)))...)
			payload = append(payload, exportName...)
			payload = append(payload, 0x00) // export kind: func
			payload = append(payload, 0x00) // function index 0
		}
		appendSection(0x07, payload)
	}

	// Code section: one empty function body.
	appendSection(0x0a, []byte{
		0x01,             // 1 body
		0x02, 0x00, 0x0b, // body size=2, no locals, end
	})

	return module
}

func encodeULEB128(v uint32) []byte {
	var out []byte
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		out = append(out, b)
		if v == 0 {
			return out
		}
	}
}
