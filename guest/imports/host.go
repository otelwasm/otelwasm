package imports

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/otelwasm/otelwasm/guest/internal/mem"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func GetConfig(v any) error {
	rawMsg := mem.GetBytes(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return getPluginConfig(ptr, limit)
	})
	return json.Unmarshal(rawMsg, v)
}

func SetResultTraces(traces ptrace.Traces) {
	marshaler := ptrace.ProtoMarshaler{}
	rawMsg, err := marshaler.MarshalTraces(traces)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	ptr, size := mem.BytesToPtr(rawMsg)
	setResultTraces(ptr, size)
	runtime.KeepAlive(rawMsg) // until ptr is no longer needed
}

func SetResultMetrics(metrics pmetric.Metrics) {
	marshaler := pmetric.ProtoMarshaler{}
	rawMsg, err := marshaler.MarshalMetrics(metrics)
	if err != nil {
		panic(err)
	}
	ptr, size := mem.BytesToPtr(rawMsg)
	setResultMetrics(ptr, size)
	runtime.KeepAlive(rawMsg) // until ptr is no longer needed
}

func SetResultLogs(logs plog.Logs) {
	marshaler := plog.ProtoMarshaler{}
	rawMsg, err := marshaler.MarshalLogs(logs)
	if err != nil {
		panic(err)
	}
	ptr, size := mem.BytesToPtr(rawMsg)
	setResultLogs(ptr, size)
	runtime.KeepAlive(rawMsg) // until ptr is no longer needed
}

// LogLevel represents the log level for structured logging
type LogLevel int32

const (
	LogLevelDebug LogLevel = 0
	LogLevelInfo  LogLevel = 1
	LogLevelWarn  LogLevel = 2
	LogLevelError LogLevel = 3
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// Log sends a structured log message to the host
func Log(level LogLevel, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Message: message,
		Fields:  fields,
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("Failed to marshal log entry: %v\n", err)
		return
	}

	ptr, size := mem.BytesToPtr(entryJSON)
	logMessage(int32(level), ptr, size)
	runtime.KeepAlive(entryJSON) // until ptr is no longer needed
}

// LogDebug logs a debug message
func LogDebug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	Log(LogLevelDebug, message, f)
}

// LogInfo logs an info message
func LogInfo(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	Log(LogLevelInfo, message, f)
}

// LogWarn logs a warning message
func LogWarn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	Log(LogLevelWarn, message, f)
}

// LogError logs an error message
func LogError(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	Log(LogLevelError, message, f)
}
