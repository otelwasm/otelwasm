//go:build !wasm

package logging

import (
	"go.uber.org/zap"
)

// NewHostBridgeLogger creates a zap.Logger for non-WASM builds
// For non-WASM builds, we just return a standard development logger
func NewHostBridgeLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		// Fallback to nop logger if development logger fails
		return zap.NewNop()
	}
	return logger
}