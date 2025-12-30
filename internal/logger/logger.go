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

// ensureLogger initializes the logger if not already set
func ensureLogger() {
	once.Do(func() {
		logger = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
	})
}

// newFileLogger creates a new file-based logger
func newFileLogger(path string) *log.Logger {
	f := must.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	return log.New(f, "", log.LstdFlags|log.Lmicroseconds)
}

// newNullLogger creates a logger that discards all output
func newNullLogger() *log.Logger {
	return newFileLogger("/dev/null")
}

// SetLogFile sets the logger to write to a file
func SetLogFile(path string) {
	logger = newFileLogger(path)
}

// SetNullLog sets the logger to discard all output
func SetNullLog() {
	logger = newNullLogger()
}

// SetStderr sets the logger to write to stderr
func SetStderr() {
	logger = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
}

// Package-level convenience functions

// Log writes a log message
func Log(format string, v ...interface{}) {
	ensureLogger()
	logger.Printf(format, v...)
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
	ensureLogger()
	logger.Printf("ERROR: "+format, v...)
}

// Info writes an info log message
func Info(format string, v ...interface{}) {
	ensureLogger()
	logger.Printf("INFO: "+format, v...)
}

// Debug writes a debug log message
func Debug(format string, v ...interface{}) {
	ensureLogger()
	logger.Printf("DEBUG: "+format, v...)
}

// Println writes a log message with newline
func Println(v ...interface{}) {
	ensureLogger()
	logger.Println(v...)
}

// Print writes a log message
func Print(v ...interface{}) {
	ensureLogger()
	logger.Print(v...)
}
