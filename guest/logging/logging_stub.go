//go:build !wasm

package logging

import (
	"log/slog"
)

// Extended log levels beyond slog to support Zap's additional levels
const (
	LevelDPanic slog.Level = slog.LevelError + 1 // 9
	LevelPanic  slog.Level = slog.LevelError + 2 // 10
	LevelFatal  slog.Level = slog.LevelError + 3 // 11
)

// LogMessage represents a structured log message to be sent to the host
type LogMessage struct {
	Level   int32             `json:"level"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

// sendLogMessage is a no-op for non-WASM builds
func sendLogMessage(level slog.Level, message string, fields map[string]string) {
	// No-op for non-WASM builds
}

// Debug logs a debug-level message
func Debug(message string, fields ...map[string]string) {
	// No-op for non-WASM builds
}

// Info logs an info-level message
func Info(message string, fields ...map[string]string) {
	// No-op for non-WASM builds
}

// Warn logs a warning-level message
func Warn(message string, fields ...map[string]string) {
	// No-op for non-WASM builds
}

// Error logs an error-level message
func Error(message string, fields ...map[string]string) {
	// No-op for non-WASM builds
}

// Logger provides a structured logging interface compatible with slog
type Logger struct{}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{}
}

// LogAttrs logs a message with structured attributes
func (l *Logger) LogAttrs(level slog.Level, msg string, attrs ...slog.Attr) {
	// No-op for non-WASM builds
}

// Debug logs a debug message with structured attributes
func (l *Logger) DebugAttrs(msg string, attrs ...slog.Attr) {
	// No-op for non-WASM builds
}

// Info logs an info message with structured attributes
func (l *Logger) InfoAttrs(msg string, attrs ...slog.Attr) {
	// No-op for non-WASM builds
}

// Warn logs a warning message with structured attributes
func (l *Logger) WarnAttrs(msg string, attrs ...slog.Attr) {
	// No-op for non-WASM builds
}

// Error logs an error message with structured attributes
func (l *Logger) ErrorAttrs(msg string, attrs ...slog.Attr) {
	// No-op for non-WASM builds
}
