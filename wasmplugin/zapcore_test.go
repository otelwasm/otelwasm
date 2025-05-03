package wasmplugin

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEntryJSONMarshalUnmarshal(t *testing.T) {
	now := time.Now().UTC()
	testCases := []struct {
		name  string
		entry Entry
	}{
		{
			name: "basic entry",
			entry: Entry{
				Level:      Level(0),
				Time:       now,
				LoggerName: "test-logger",
				Message:    "test message",
				Caller: EntryCaller{
					Defined:  true,
					PC:       0x1234,
					File:     "zapcore_test.go",
					Line:     42,
					Function: "TestFunction",
				},
				Stack: "test stack trace",
			},
		},
		{
			name: "entry with empty caller",
			entry: Entry{
				Level:      Level(1),
				Time:       now,
				LoggerName: "another-logger",
				Message:    "another message",
				Caller:     EntryCaller{},
				Stack:      "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal to JSON
			jsonData, err := json.Marshal(tc.entry)
			assert.NoError(t, err)
			assert.NotEmpty(t, jsonData)

			// Unmarshal from JSON
			var unmarshaled Entry
			err = json.Unmarshal(jsonData, &unmarshaled)
			assert.NoError(t, err)

			// Compare fields
			assert.Equal(t, tc.entry.Level, unmarshaled.Level)
			assert.Equal(t, tc.entry.Time.Unix(), unmarshaled.Time.Unix())
			assert.Equal(t, tc.entry.LoggerName, unmarshaled.LoggerName)
			assert.Equal(t, tc.entry.Message, unmarshaled.Message)
			assert.Equal(t, tc.entry.Caller.Defined, unmarshaled.Caller.Defined)
			assert.Equal(t, tc.entry.Caller.PC, unmarshaled.Caller.PC)
			assert.Equal(t, tc.entry.Caller.File, unmarshaled.Caller.File)
			assert.Equal(t, tc.entry.Caller.Line, unmarshaled.Caller.Line)
			assert.Equal(t, tc.entry.Caller.Function, unmarshaled.Caller.Function)
			assert.Equal(t, tc.entry.Stack, unmarshaled.Stack)
		})
	}
}

func TestEntryCallerJSONMarshalUnmarshal(t *testing.T) {
	testCases := []struct {
		name   string
		caller EntryCaller
	}{
		{
			name: "defined caller",
			caller: EntryCaller{
				Defined:  true,
				PC:       0x5678,
				File:     "some_file.go",
				Line:     123,
				Function: "SomeFunction",
			},
		},
		{
			name:   "undefined caller",
			caller: EntryCaller{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal to JSON
			jsonData, err := json.Marshal(tc.caller)
			assert.NoError(t, err)
			assert.NotEmpty(t, jsonData)

			// Unmarshal from JSON
			var unmarshaled EntryCaller
			err = json.Unmarshal(jsonData, &unmarshaled)
			assert.NoError(t, err)

			// Compare fields
			assert.Equal(t, tc.caller, unmarshaled)
		})
	}
}
