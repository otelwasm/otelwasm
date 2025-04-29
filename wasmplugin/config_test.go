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
			name: "valid interpreter mode",
			config: RuntimeConfig{
				Mode: RuntimeModeInterpreter,
			},
			wantErr: false,
		},
		{
			name: "valid compiled mode",
			config: RuntimeConfig{
				Mode: RuntimeModeCompiled,
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: RuntimeConfig{
				Mode: "invalid",
			},
			wantErr: true,
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
		name           string
		config         RuntimeConfig
		expectedConfig RuntimeConfig
	}{
		{
			name:           "empty mode",
			config:         RuntimeConfig{},
			expectedConfig: RuntimeConfig{Mode: RuntimeModeInterpreter},
		},
		{
			name:           "compiled mode",
			config:         RuntimeConfig{Mode: RuntimeModeCompiled},
			expectedConfig: RuntimeConfig{Mode: RuntimeModeCompiled},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.Default()
			if tt.config.Mode != tt.expectedConfig.Mode {
				t.Errorf("RuntimeConfig.Default() set Mode = %v, want %v", tt.config.Mode, tt.expectedConfig.Mode)
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
				RuntimeConfig: RuntimeConfig{
					Mode: RuntimeModeInterpreter,
				},
			},
			wantErr: false,
		},
		{
			name: "missing path",
			config: Config{
				RuntimeConfig: RuntimeConfig{
					Mode: RuntimeModeInterpreter,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid runtime mode",
			config: Config{
				Path: "test.wasm",
				RuntimeConfig: RuntimeConfig{
					Mode: "invalid",
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
				RuntimeConfig: RuntimeConfig{
					Mode: RuntimeModeCompiled,
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
		name           string
		config         Config
		expectedConfig Config
	}{
		{
			name: "empty config",
			config: Config{
				Path: "test.wasm",
			},
			expectedConfig: Config{
				Path: "test.wasm",
				RuntimeConfig: RuntimeConfig{
					Mode: RuntimeModeInterpreter,
				},
			},
		},
		{
			name: "config with custom mode",
			config: Config{
				Path: "test.wasm",
				RuntimeConfig: RuntimeConfig{
					Mode: RuntimeModeCompiled,
				},
			},
			expectedConfig: Config{
				Path: "test.wasm",
				RuntimeConfig: RuntimeConfig{
					Mode: RuntimeModeCompiled,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.Default()
			if tt.config.RuntimeConfig.Mode != tt.expectedConfig.RuntimeConfig.Mode {
				t.Errorf("Config.Default() set RuntimeConfig.Mode = %v, want %v",
					tt.config.RuntimeConfig.Mode, tt.expectedConfig.RuntimeConfig.Mode)
			}
		})
	}
}
