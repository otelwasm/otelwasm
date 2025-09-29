package wasmplugin

import (
	"testing"
)

func TestRuntimeConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  RuntimeConfig
		wantErr bool
	}{
		{
			name: "valid wazero runtime with interpreter mode",
			config: RuntimeConfig{
				Type: RuntimeTypeWazero,
				Wazero: &WazeroConfig{
					Mode: WazeroRuntimeModeInterpreter,
				},
			},
			wantErr: false,
		},
		{
			name: "valid wazero runtime with compiled mode",
			config: RuntimeConfig{
				Type: RuntimeTypeWazero,
				Wazero: &WazeroConfig{
					Mode: WazeroRuntimeModeCompiled,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid runtime type",
			config: RuntimeConfig{
				Type: "invalid",
			},
			wantErr: true,
		},
		{
			name:    "empty runtime type (should default to wazero)",
			config:  RuntimeConfig{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RuntimeConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuntimeConfigDefault(t *testing.T) {
	tests := []struct {
		name         string
		config       RuntimeConfig
		expectedType string
		expectedMode WazeroRuntimeMode
	}{
		{
			name:         "empty config",
			config:       RuntimeConfig{},
			expectedType: RuntimeTypeWazero,
			expectedMode: WazeroRuntimeModeInterpreter,
		},
		{
			name: "explicit wazero compiled mode",
			config: RuntimeConfig{
				Type:   RuntimeTypeWazero,
				Wazero: &WazeroConfig{Mode: WazeroRuntimeModeCompiled},
			},
			expectedType: RuntimeTypeWazero,
			expectedMode: WazeroRuntimeModeCompiled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.Default()
			if tt.config.Type != tt.expectedType {
				t.Errorf("RuntimeConfig.Default() set Type = %v, want %v", tt.config.Type, tt.expectedType)
			}
			if tt.config.Wazero != nil && tt.config.Wazero.Mode != tt.expectedMode {
				t.Errorf("RuntimeConfig.Default() set Wazero.Mode = %v, want %v", tt.config.Wazero.Mode, tt.expectedMode)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Path: "test.wasm",
				RuntimeConfig: &RuntimeConfig{
					Type: RuntimeTypeWazero,
					Wazero: &WazeroConfig{
						Mode: WazeroRuntimeModeInterpreter,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing path",
			config: Config{
				RuntimeConfig: &RuntimeConfig{
					Type: RuntimeTypeWazero,
					Wazero: &WazeroConfig{
						Mode: WazeroRuntimeModeInterpreter,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid runtime type",
			config: Config{
				Path: "test.wasm",
				RuntimeConfig: &RuntimeConfig{
					Type: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "valid config with plugin config",
			config: Config{
				Path: "test.wasm",
				PluginConfig: PluginConfig{
					"key": "value",
				},
				RuntimeConfig: &RuntimeConfig{
					Type: RuntimeTypeWazero,
					Wazero: &WazeroConfig{
						Mode: WazeroRuntimeModeCompiled,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigDefault(t *testing.T) {
	tests := []struct {
		name         string
		config       Config
		expectedType string
		expectedMode WazeroRuntimeMode
	}{
		{
			name: "empty config",
			config: Config{
				Path: "test.wasm",
			},
			expectedType: RuntimeTypeWazero,
			expectedMode: WazeroRuntimeModeInterpreter,
		},
		{
			name: "config with custom mode",
			config: Config{
				Path: "test.wasm",
				RuntimeConfig: &RuntimeConfig{
					Type: RuntimeTypeWazero,
					Wazero: &WazeroConfig{
						Mode: WazeroRuntimeModeCompiled,
					},
				},
			},
			expectedType: RuntimeTypeWazero,
			expectedMode: WazeroRuntimeModeCompiled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.Default()
			if tt.config.RuntimeConfig.Type != tt.expectedType {
				t.Errorf("Config.Default() set RuntimeConfig.Type = %v, want %v",
					tt.config.RuntimeConfig.Type, tt.expectedType)
			}
			if tt.config.RuntimeConfig.Wazero != nil && tt.config.RuntimeConfig.Wazero.Mode != tt.expectedMode {
				t.Errorf("Config.Default() set RuntimeConfig.Wazero.Mode = %v, want %v",
					tt.config.RuntimeConfig.Wazero.Mode, tt.expectedMode)
			}
		})
	}
}
