package logger

import (
	"log"
	"os"
	"sync"

	"git.15b.it/eno/critic/internal/must"
)

var (
	logger *log.Logger
	once   sync.Once
)

// NewFileLogger creates a new file-based logger
func NewFileLogger(path string) *log.Logger {
	f := must.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	return log.New(f, "", log.LstdFlags|log.Lmicroseconds)
}

// NullLogger is a logger that discards all output
type NullLogger struct{}

// NewNullLogger creates a new null logger
func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

// Info does nothing
func (l *NullLogger) Info(format string, v ...interface{}) {}

// Error does nothing
func (l *NullLogger) Error(format string, v ...interface{}) {}

// Debug does nothing
func (l *NullLogger) Debug(format string, v ...interface{}) {}

// Log does nothing
func (l *NullLogger) Log(format string, v ...interface{}) {}

// Print does nothing
func (l *NullLogger) Print(v ...interface{}) {}

// Println does nothing
func (l *NullLogger) Println(v ...interface{}) {}

// Init initializes the package-level logger
func Init() {
	once.Do(func() {
		logger = NewFileLogger("/tmp/critic.log")
	})
}

// Package-level convenience functions

// Log writes a log message
func Log(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf(format, v...)
	}
}

// Logf is an alias for Log
func Logf(format string, v ...interface{}) {
	Log(format, v...)
}

// Printf writes a log message (compatibility)
func Printf(format string, v ...interface{}) {
	Log(format, v...)
}

// Error writes an error log message
func Error(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("ERROR: "+format, v...)
	}
}

// Info writes an info log message
func Info(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("INFO: "+format, v...)
	}
}

// Debug writes a debug log message
func Debug(format string, v ...interface{}) {
	if logger != nil {
		logger.Printf("DEBUG: "+format, v...)
	}
}

// Println writes a log message with newline
func Println(v ...interface{}) {
	if logger != nil {
		logger.Println(v...)
	}
}

// Print writes a log message
func Print(v ...interface{}) {
	if logger != nil {
		logger.Print(v...)
	}
}
