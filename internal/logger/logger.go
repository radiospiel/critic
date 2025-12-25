package logger

import (
	"io"
	"log"
	"os"
	"sync"
)

// Logger is an interface for logging operations
type Logger interface {
	Info(format string, v ...interface{})
	Error(format string, v ...interface{})
	Debug(format string, v ...interface{})
	Log(format string, v ...interface{})
	Print(v ...interface{})
	Println(v ...interface{})
}

var (
	defaultLogger Logger
	once          sync.Once
	mu            sync.RWMutex
)

// FileLogger implements Logger interface with file output
type FileLogger struct {
	logger *log.Logger
	file   *os.File
}

// NewFileLogger creates a new file-based logger
func NewFileLogger(path string) (*FileLogger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &FileLogger{
		logger: log.New(f, "", log.LstdFlags|log.Lmicroseconds),
		file:   f,
	}, nil
}

// NewFileLoggerWithWriter creates a logger with a custom writer
func NewFileLoggerWithWriter(w io.Writer) *FileLogger {
	return &FileLogger{
		logger: log.New(w, "", log.LstdFlags|log.Lmicroseconds),
		file:   nil, // No file to close
	}
}

// Close closes the log file
func (l *FileLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Info logs an info message
func (l *FileLogger) Info(format string, v ...interface{}) {
	l.logger.Printf("INFO: "+format, v...)
}

// Error logs an error message
func (l *FileLogger) Error(format string, v ...interface{}) {
	l.logger.Printf("ERROR: "+format, v...)
}

// Debug logs a debug message
func (l *FileLogger) Debug(format string, v ...interface{}) {
	l.logger.Printf("DEBUG: "+format, v...)
}

// Log logs a message
func (l *FileLogger) Log(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}

// Print logs a message
func (l *FileLogger) Print(v ...interface{}) {
	l.logger.Print(v...)
}

// Println logs a message with newline
func (l *FileLogger) Println(v ...interface{}) {
	l.logger.Println(v...)
}

// NullLogger is a logger that discards all output (for testing)
type NullLogger struct{}

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

// Init initializes the default logger
func Init() error {
	var err error
	once.Do(func() {
		l, e := NewFileLogger("/tmp/critic.log")
		if e != nil {
			err = e
			return
		}
		SetLogger(l)
	})
	return err
}

// SetLogger sets the logger instance (useful for testing)
func SetLogger(l Logger) {
	mu.Lock()
	defer mu.Unlock()
	defaultLogger = l
}

// getLogger returns the current logger instance
func getLogger() Logger {
	mu.RLock()
	defer mu.RUnlock()
	return defaultLogger
}

// Package-level convenience functions that delegate to the default logger

// Log writes a log message
func Log(format string, v ...interface{}) {
	if l := getLogger(); l != nil {
		l.Log(format, v...)
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
	if l := getLogger(); l != nil {
		l.Error(format, v...)
	}
}

// Info writes an info log message
func Info(format string, v ...interface{}) {
	if l := getLogger(); l != nil {
		l.Info(format, v...)
	}
}

// Debug writes a debug log message
func Debug(format string, v ...interface{}) {
	if l := getLogger(); l != nil {
		l.Debug(format, v...)
	}
}

// Println writes a log message with newline
func Println(v ...interface{}) {
	if l := getLogger(); l != nil {
		l.Println(v...)
	}
}

// Print writes a log message
func Print(v ...interface{}) {
	if l := getLogger(); l != nil {
		l.Print(v...)
	}
}
