//go:build wasm

package logging

import (
	"encoding/json"
	"log/slog"
	"runtime"

	"github.com/otelwasm/otelwasm/guest/internal/mem"
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

//go:wasmimport opentelemetry.io/wasm logMessage
func logMessage(ptr, size uint32)

// sendLogMessage sends a log message to the host
func sendLogMessage(level slog.Level, message string, fields map[string]string) {
	logMsg := LogMessage{
		Level:   int32(level),
		Message: message,
		Fields:  fields,
	}

	// Marshal the log message
	logBytes, err := json.Marshal(logMsg)
	if err != nil {
		// If marshaling fails, we can't log it, so we return silently
		return
	}

	// Send to host via WASM memory
	ptr, size := mem.BytesToPtr(logBytes)
	logMessage(ptr, size)
	runtime.KeepAlive(logBytes) // until ptr is no longer needed
}

// Debug logs a debug-level message
func Debug(message string, fields ...map[string]string) {
	var fieldMap map[string]string
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	sendLogMessage(slog.LevelDebug, message, fieldMap)
}

// Info logs an info-level message
func Info(message string, fields ...map[string]string) {
	var fieldMap map[string]string
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	sendLogMessage(slog.LevelInfo, message, fieldMap)
}

// Warn logs a warning-level message
func Warn(message string, fields ...map[string]string) {
	var fieldMap map[string]string
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	sendLogMessage(slog.LevelWarn, message, fieldMap)
}

// Error logs an error-level message
func Error(message string, fields ...map[string]string) {
	var fieldMap map[string]string
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	sendLogMessage(slog.LevelError, message, fieldMap)
}

// Logger provides a structured logging interface compatible with slog
type Logger struct{}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{}
}

// LogAttrs logs a message with structured attributes
func (l *Logger) LogAttrs(level slog.Level, msg string, attrs ...slog.Attr) {
	fields := make(map[string]string)
	for _, attr := range attrs {
		fields[attr.Key] = attr.Value.String()
	}
	sendLogMessage(level, msg, fields)
}

// Debug logs a debug message with structured attributes
func (l *Logger) DebugAttrs(msg string, attrs ...slog.Attr) {
	l.LogAttrs(slog.LevelDebug, msg, attrs...)
}

// Info logs an info message with structured attributes
func (l *Logger) InfoAttrs(msg string, attrs ...slog.Attr) {
	l.LogAttrs(slog.LevelInfo, msg, attrs...)
}

// Warn logs a warning message with structured attributes
func (l *Logger) WarnAttrs(msg string, attrs ...slog.Attr) {
	l.LogAttrs(slog.LevelWarn, msg, attrs...)
}

// Error logs an error message with structured attributes
func (l *Logger) ErrorAttrs(msg string, attrs ...slog.Attr) {
	l.LogAttrs(slog.LevelError, msg, attrs...)
}