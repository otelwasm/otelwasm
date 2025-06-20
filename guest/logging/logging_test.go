//go:build !wasm

package logging

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogMessage(t *testing.T) {
	// Test that LogMessage struct can be created and marshaled
	logMsg := LogMessage{
		Level:   int32(slog.LevelInfo),
		Message: "test message",
		Fields:  map[string]string{"key": "value"},
	}

	assert.Equal(t, int32(slog.LevelInfo), logMsg.Level)
	assert.Equal(t, "test message", logMsg.Message)
	assert.Equal(t, "value", logMsg.Fields["key"])
}

func TestLoggerMethods(t *testing.T) {
	// Since these are no-op for non-WASM builds, we just test they don't panic
	logger := NewLogger()
	assert.NotNil(t, logger)

	// Test global functions
	assert.NotPanics(t, func() {
		Debug("debug message")
		Info("info message")
		Warn("warn message")
		Error("error message")
	})

	// Test with fields
	fields := map[string]string{"key": "value"}
	assert.NotPanics(t, func() {
		Debug("debug with fields", fields)
		Info("info with fields", fields)
		Warn("warn with fields", fields)
		Error("error with fields", fields)
	})

	// Test logger methods
	assert.NotPanics(t, func() {
		logger.DebugAttrs("debug attrs")
		logger.InfoAttrs("info attrs")
		logger.WarnAttrs("warn attrs")
		logger.ErrorAttrs("error attrs")
	})

	// Test LogAttrs
	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	}
	assert.NotPanics(t, func() {
		logger.LogAttrs(slog.LevelInfo, "test message", attrs...)
	})
}