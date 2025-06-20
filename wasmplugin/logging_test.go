package wasmplugin

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogMessageFn(t *testing.T) {
	tests := []struct {
		name           string
		logMessage     LogMessage
		expectedLevel  zapcore.Level
		expectedMsg    string
		expectedFields map[string]string
	}{
		{
			name: "debug message",
			logMessage: LogMessage{
				Level:   int32(-4), // slog.LevelDebug
				Message: "debug message",
				Fields:  map[string]string{"key1": "value1"},
			},
			expectedLevel:  zapcore.DebugLevel,
			expectedMsg:    "debug message",
			expectedFields: map[string]string{"key1": "value1"},
		},
		{
			name: "info message",
			logMessage: LogMessage{
				Level:   int32(0), // slog.LevelInfo
				Message: "info message",
				Fields:  map[string]string{"key2": "value2"},
			},
			expectedLevel:  zapcore.InfoLevel,
			expectedMsg:    "info message",
			expectedFields: map[string]string{"key2": "value2"},
		},
		{
			name: "warn message",
			logMessage: LogMessage{
				Level:   int32(4), // slog.LevelWarn
				Message: "warn message",
				Fields:  map[string]string{"key3": "value3"},
			},
			expectedLevel:  zapcore.WarnLevel,
			expectedMsg:    "warn message",
			expectedFields: map[string]string{"key3": "value3"},
		},
		{
			name: "error message",
			logMessage: LogMessage{
				Level:   int32(8), // slog.LevelError
				Message: "error message",
				Fields:  map[string]string{"key4": "value4"},
			},
			expectedLevel:  zapcore.ErrorLevel,
			expectedMsg:    "error message",
			expectedFields: map[string]string{"key4": "value4"},
		},
		{
			name: "message without fields",
			logMessage: LogMessage{
				Level:   int32(0), // slog.LevelInfo
				Message: "simple message",
				Fields:  nil,
			},
			expectedLevel:  zapcore.InfoLevel,
			expectedMsg:    "simple message",
			expectedFields: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create an observed logger
			core, observed := observer.New(zapcore.DebugLevel)
			logger := zap.New(core)

			// Create a context with a stack containing the logger
			stack := &Stack{Logger: logger}
			ctx := createContextWithStack(context.Background(), stack)

			// Create a runtime and module for testing
			runtime := wazero.NewRuntime(ctx)
			defer runtime.Close(ctx)

			// Create a module with memory
			mod, err := runtime.Instantiate(ctx, []byte(`
				(module
					(memory (export "memory") 1)
				)
			`))
			require.NoError(t, err)
			defer mod.Close(ctx)

			// Marshal the log message to JSON
			logBytes, err := json.Marshal(tt.logMessage)
			require.NoError(t, err)

			// Write the log message to module memory
			require.True(t, mod.Memory().Write(0, logBytes))

			// Call the log message function
			wasmStack := []uint64{0, uint64(len(logBytes))}
			logMessageFn(ctx, mod, wasmStack)

			// Verify the log was written
			logs := observed.All()
			require.Len(t, logs, 1)

			log := logs[0]
			assert.Equal(t, tt.expectedLevel, log.Level)
			assert.Equal(t, tt.expectedMsg, log.Message)

			// Check the fields
			expectedFieldCount := len(tt.expectedFields)
			assert.Len(t, log.Context, expectedFieldCount)

			for key, expectedValue := range tt.expectedFields {
				found := false
				for _, field := range log.Context {
					if field.Key == key {
						assert.Equal(t, expectedValue, field.String)
						found = true
						break
					}
				}
				assert.True(t, found, "field %s not found", key)
			}
		})
	}
}

func TestLogMessageFnWithInvalidJSON(t *testing.T) {
	// Create an observed logger
	core, observed := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	// Create a context with a stack containing the logger
	stack := &Stack{Logger: logger}
	ctx := createContextWithStack(context.Background(), stack)

	// Create a runtime and module for testing
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	// Create a module with memory
	mod, err := runtime.Instantiate(ctx, []byte(`
		(module
			(memory (export "memory") 1)
		)
	`))
	require.NoError(t, err)
	defer mod.Close(ctx)

	// Write invalid JSON to module memory
	invalidJSON := []byte(`{"invalid": json}`)
	require.True(t, mod.Memory().Write(0, invalidJSON))

	// Call the log message function
	wasmStack := []uint64{0, uint64(len(invalidJSON))}
	logMessageFn(ctx, mod, wasmStack)

	// Verify an error log was written about the unmarshal failure
	logs := observed.All()
	require.Len(t, logs, 1)

	log := logs[0]
	assert.Equal(t, zapcore.ErrorLevel, log.Level)
	assert.Contains(t, log.Message, "failed to unmarshal log message from guest")
}

func TestLogMessageFnWithoutLogger(t *testing.T) {
	// Create a context with a stack that has no logger
	stack := &Stack{Logger: nil}
	ctx := createContextWithStack(context.Background(), stack)

	// Create a runtime and module for testing
	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	// Create a module with memory
	mod, err := runtime.Instantiate(ctx, []byte(`
		(module
			(memory (export "memory") 1)
		)
	`))
	require.NoError(t, err)
	defer mod.Close(ctx)

	// Create a valid log message
	logMessage := LogMessage{
		Level:   int32(0),
		Message: "test message",
		Fields:  map[string]string{"key": "value"},
	}

	logBytes, err := json.Marshal(logMessage)
	require.NoError(t, err)

	// Write the log message to module memory
	require.True(t, mod.Memory().Write(0, logBytes))

	// Call the log message function - this should not panic
	wasmStack := []uint64{0, uint64(len(logBytes))}
	assert.NotPanics(t, func() {
		logMessageFn(ctx, mod, wasmStack)
	})
}

func TestZapLevelFromSlogLevel(t *testing.T) {
	tests := []struct {
		name      string
		slogLevel int32
		expected  zapcore.Level
	}{
		{"debug level", -4, zapcore.DebugLevel},
		{"info level", 0, zapcore.InfoLevel},
		{"warn level", 4, zapcore.WarnLevel},
		{"error level", 8, zapcore.ErrorLevel},
		{"very high level", 100, zapcore.ErrorLevel},
		{"very low level", -100, zapcore.DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert int32 to slog.Level and then to zapcore.Level
			level := zapLevelFromSlogLevel(slog.Level(tt.slogLevel))
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestTelemetrySettingsToSerializable(t *testing.T) {
	// Create test telemetry settings
	ts := component.TelemetrySettings{
		Resource: pcommon.NewResource(),
	}

	// Add some resource attributes
	attrs := ts.Resource.Attributes()
	attrs.PutStr("service.name", "test-service")
	attrs.PutStr("service.version", "1.0.0")
	attrs.PutInt("test.number", 42)
	attrs.PutBool("test.bool", true)
	attrs.PutDouble("test.double", 3.14)

	// Convert to serializable
	serializable := telemetrySettingsToSerializable(ts)

	// Verify the conversion
	assert.Equal(t, "test-service", serializable.ServiceName)
	assert.Equal(t, "1.0.0", serializable.ServiceVersion)
	assert.Equal(t, "test-service", serializable.ResourceAttributes["service.name"])
	assert.Equal(t, "1.0.0", serializable.ResourceAttributes["service.version"])
	assert.Equal(t, int64(42), serializable.ResourceAttributes["test.number"])
	assert.Equal(t, true, serializable.ResourceAttributes["test.bool"])
	assert.Equal(t, 3.14, serializable.ResourceAttributes["test.double"])
	assert.NotNil(t, serializable.ComponentID)
}

func TestTelemetrySettingsToSerializableEmpty(t *testing.T) {
	// Create empty telemetry settings
	ts := component.TelemetrySettings{
		Resource: pcommon.NewResource(),
	}

	// Convert to serializable
	serializable := telemetrySettingsToSerializable(ts)

	// Verify the conversion
	assert.Empty(t, serializable.ServiceName)
	assert.Empty(t, serializable.ServiceVersion)
	assert.NotNil(t, serializable.ResourceAttributes)
	assert.Empty(t, serializable.ResourceAttributes)
	assert.NotNil(t, serializable.ComponentID)
	assert.Empty(t, serializable.ComponentID)
}