package wasmplugin

import (
	"encoding/json"
	"testing"
)

func TestLogLevel_String(t *testing.T) {
	testCases := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tc := range testCases {
		if got := tc.level.String(); got != tc.expected {
			t.Errorf("LogLevel(%d).String() = %s, want %s", tc.level, got, tc.expected)
		}
	}
}

func TestLogEntry_Marshal(t *testing.T) {
	entry := LogEntry{
		Message: "test message",
		Fields: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	// Test that we can marshal and unmarshal the LogEntry
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal LogEntry: %v", err)
	}

	var unmarshaled LogEntry
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal LogEntry: %v", err)
	}

	if unmarshaled.Message != entry.Message {
		t.Errorf("Expected message %s, got %s", entry.Message, unmarshaled.Message)
	}

	if len(unmarshaled.Fields) != len(entry.Fields) {
		t.Errorf("Expected %d fields, got %d", len(entry.Fields), len(unmarshaled.Fields))
	}
}