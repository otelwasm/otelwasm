package wasmplugin

import (
	"time"

	"go.uber.org/zap/zapcore"
)

// Level is wrapper interface for zapcore.Level
type Level int8

// EntryCaller is wrapper interface for zapcore.EntryCaller
type EntryCaller struct {
	Defined  bool    `json:"defined"`
	PC       uintptr `json:"pc"`
	File     string  `json:"file"`
	Line     int     `json:"line"`
	Function string  `json:"function"`
}

// Entry is wrapper interface for zapcore.Entry
type Entry struct {
	Level      Level       `json:"level"`
	Time       time.Time   `json:"time"`
	LoggerName string      `json:"logger_name"`
	Message    string      `json:"message"`
	Caller     EntryCaller `json:"caller"`
	Stack      string      `json:"stack"`
}

// FieldType is wrapper interface for zapcore.FieldType
type FieldType uint8

// Field is wrapper interface for zapcore.Field
type Field struct {
	Key     string
	Type    FieldType
	Integer int64
	String  string
	// TODO: Add the following field after checking how to properly handle it.
	// Interface interface{}
}

type Fields []Field

func (f Fields) ZapCoreFields() []zapcore.Field {
	zapFields := make([]zapcore.Field, len(f))
	for i, field := range f {
		zapFields[i] = zapcore.Field{
			Key:     field.Key,
			Type:    zapcore.FieldType(field.Type),
			Integer: field.Integer,
			String:  field.String,
		}
	}
	return zapFields
}
