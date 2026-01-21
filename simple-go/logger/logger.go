package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level represents a log level
type Level int

const enableStdoutLog = false
const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	once   sync.Once
	level  Level = INFO // default log level
	prefix string
)

// stores the path to the current log file
var logFilePath = buildLogFilePath()
var fileLogger = newFileLogger(logFilePath)

// defaultLogger is a TopicLogger with empty topic (no prefix)
var defaultLogger = &TopicLogger{topic: ""}

func SetPrefix(s string) {
	prefix = s
}

func Runtime[T any](msg string, fun func() T) T {
	start := time.Now()
	r := fun()

	durationMs := float64(time.Since(start).Microseconds()) / 1000.0
	Info("%s: %.1f msecs", msg, durationMs)
	return r
}

// buildLogFilePath determines the log file path from the process name
func buildLogFilePath() string {
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}

	// Get the process name from os.Args[0]
	processName := filepath.Base(os.Args[0])

	return filepath.Join(tmpDir, processName+".log")
}

// LogFilePath returns the full path to the current log file
func LogFilePath() string {
	return logFilePath
}

// newFileLogger creates a new file-based logger
func newFileLogger(path string) *log.Logger {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("OpenFile(%s) failed: %s", path, err))
	}

	return log.New(f, "", log.LstdFlags|log.Lmicroseconds)
}

// newNullLogger creates a logger that discards all output
func newNullLogger() *log.Logger {
	return newFileLogger("/dev/null")
}

// SetLogFile sets the logger to write to a file
func SetLogFile(path string) {
	logFilePath = path
	fileLogger = newFileLogger(path)
}

// SetNullLog sets the logger to discard all output
func SetNullLog() {
	fileLogger = newNullLogger()
}

// SetLevel sets the minimum log level
func SetLevel(l Level) {
	level = l
}

// TopicLogger is a logger that prepends a topic tag to all log messages
type TopicLogger struct {
	topic string
}

// OnTopic returns a TopicLogger that prepends [topic] to all log messages
func OnTopic(topic string) *TopicLogger {
	return &TopicLogger{topic: topic}
}

// Printf writes a log message with optional topic prefix
func (t *TopicLogger) Printf(format string, v ...interface{}) {
	var finalFormat string
	var finalArgs []interface{}

	if t.topic != "" {
		finalFormat = "[%s] " + format
		finalArgs = append([]interface{}{t.topic}, v...)
	} else {
		finalFormat = format
		finalArgs = v
	}

	go func() {
		fileLogger.Printf(finalFormat, finalArgs...)
		if enableStdoutLog {
			if prefix != "" {
				finalArgs = append([]interface{}{prefix}, finalArgs...)
				fmt.Printf("%s "+finalFormat+"\n", finalArgs...)
			} else {
				fmt.Printf(finalFormat+"\n", finalArgs...)
			}
		}
	}()
}

// Error writes an error log message with topic prefix
func (t *TopicLogger) Error(format string, v ...interface{}) {
	if level > ERROR {
		return
	}
	t.Printf("ERROR: "+format, v...)
}

// Warn writes a warning log message with topic prefix
func (t *TopicLogger) Warn(format string, v ...interface{}) {
	if level > WARN {
		return
	}
	t.Printf("WARN: "+format, v...)
}

// Fatal writes a fatal error log message with topic prefix and exits
func (t *TopicLogger) Fatal(format string, v ...interface{}) {
	t.Printf("FATAL: "+format, v...)
	os.Exit(1)
}

// Info writes an info log message with topic prefix
func (t *TopicLogger) Info(format string, v ...interface{}) {
	if level > INFO {
		return
	}
	t.Printf("INFO: "+format, v...)
}

// Debug writes a debug log message with topic prefix
func (t *TopicLogger) Debug(format string, v ...interface{}) {
	if level > DEBUG {
		return
	}
	t.Printf("DEBUG: "+format, v...)
}

// Package-level convenience functions that delegate to defaultLogger

// Printf writes a log message
func Printf(format string, v ...interface{}) {
	defaultLogger.Printf(format, v...)
}

// Error writes an error log message
func Error(format string, v ...interface{}) {
	defaultLogger.Error(format, v...)
}

// Warn writes a warning log message
func Warn(format string, v ...interface{}) {
	defaultLogger.Warn(format, v...)
}

// Fatal writes a fatal error log message and exits
func Fatal(format string, v ...interface{}) {
	defaultLogger.Fatal(format, v...)
}

// Info writes an info log message
func Info(format string, v ...interface{}) {
	defaultLogger.Info(format, v...)
}

// Debug writes a debug log message
func Debug(format string, v ...interface{}) {
	defaultLogger.Debug(format, v...)
}
