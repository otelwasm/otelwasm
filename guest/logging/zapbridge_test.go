//go:build !wasm

package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewHostBridgeLogger(t *testing.T) {
	// Test that the bridge logger is created successfully
	logger := NewHostBridgeLogger()
	assert.NotNil(t, logger)

	// For non-WASM builds, this should be a real zap logger
	assert.IsType(t, &zap.Logger{}, logger)

	// Test that we can log with it (should not panic)
	assert.NotPanics(t, func() {
		logger.Info("test message")
		logger.Debug("debug message")
		logger.Warn("warning message")
		logger.Error("error message")
	})
}

func TestZapBridgeWithFields(t *testing.T) {
	logger := NewHostBridgeLogger()
	
	// Test logging with fields (should not panic)
	assert.NotPanics(t, func() {
		logger.Info("test with fields", 
			zap.String("key1", "value1"),
			zap.Int("key2", 42),
			zap.Bool("key3", true),
		)
	})
}