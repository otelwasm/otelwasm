//go:build wasm

package logging

import (
	"log/slog"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewHostBridgeLogger creates a zap.Logger that bridges to the host-side logger
// through our logging host function. This eliminates the need for guest code
// to create its own zap logger.
func NewHostBridgeLogger() *zap.Logger {
	// Create a custom zapcore.Core that forwards to our host logging
	core := &hostBridgeCore{}
	return zap.New(core)
}

// hostBridgeCore implements zapcore.Core and forwards all log entries
// to the host-side logger through our logging host function
type hostBridgeCore struct{}

// Enabled always returns true since we want to let the host decide filtering
func (c *hostBridgeCore) Enabled(level zapcore.Level) bool {
	return true
}

// With returns a new core with the given fields added
func (c *hostBridgeCore) With(fields []zapcore.Field) zapcore.Core {
	// For simplicity, we return the same core since we handle fields differently
	return c
}

// Check determines whether the supplied Entry should be logged
func (c *hostBridgeCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checkedEntry.AddCore(entry, c)
	}
	return checkedEntry
}

// Write serializes the Entry and any Fields supplied at the log site and writes them to the host
func (c *hostBridgeCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Convert zap level to our extended slog level
	var slogLevel slog.Level
	switch entry.Level {
	case zapcore.DebugLevel:
		slogLevel = slog.LevelDebug
	case zapcore.InfoLevel:
		slogLevel = slog.LevelInfo
	case zapcore.WarnLevel:
		slogLevel = slog.LevelWarn
	case zapcore.ErrorLevel:
		slogLevel = slog.LevelError
	case zapcore.DPanicLevel:
		slogLevel = LevelDPanic
	case zapcore.PanicLevel:
		slogLevel = LevelPanic
	case zapcore.FatalLevel:
		slogLevel = LevelFatal
	default:
		slogLevel = slog.LevelInfo
	}

	// Convert zap fields to map
	fieldMap := make(map[string]string)
	for _, field := range fields {
		fieldMap[field.Key] = field.String
	}

	// Add entry context fields
	if entry.LoggerName != "" {
		fieldMap["logger"] = entry.LoggerName
	}
	if entry.Caller.Defined {
		fieldMap["caller"] = entry.Caller.String()
	}

	// Send to host logger
	sendLogMessage(slogLevel, entry.Message, fieldMap)
	return nil
}

// Sync flushes buffered logs (no-op for our implementation)
func (c *hostBridgeCore) Sync() error {
	return nil
}