package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Inspect is an interface for custom log formatting.
// Types implementing this interface will use InspectForLog() for %v formatting in log messages.
type Inspect interface {
	InspectForLog() string
}

// Level represents a log minLogLevel
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	minLogLevel Level = INFO // default log minLogLevel
)

// stores the path to the current log file
var logFilePath = buildLogFilePath()
var fileLogger = newFileLogger(logFilePath)

// logMessage holds the data for a log entry
type logMessage struct {
	file string
	line int

	topic  string
	format string
	args   []any
}

// logChannel is a buffered channel for log messages
var logChannel = make(chan logMessage, 1000)

var wd = (func() string { dir, _ := os.Getwd(); return dir })()

// transformArgs converts any Inspect-implementing values to their string representation
func transformArgs(args []any) []any {
	result := make([]any, len(args))
	for i, arg := range args {
		if inspector, ok := arg.(Inspect); ok {
			result[i] = inspector.InspectForLog()
		} else {
			result[i] = arg
		}
	}
	return result
}

func init() {
	// Start background writer goroutine
	go func() {
		for entry := range logChannel {
			args := transformArgs(entry.args)
			msg := fmt.Sprintf(entry.format, args...)
			if entry.topic != "" {
				msg = "[" + entry.topic + "]: " + msg
			}
			if entry.file != "" && entry.line != 0 {
				file := entry.file
				if file[0:len(wd)] == wd {
					file = file[len(wd)+1:]
				}
				msg = fmt.Sprintf("%s(%d): ", file, entry.line) + msg
			}
			fileLogger.Println(msg)
		}
	}()
}

// defaultLogger is a TopicLogger with empty topic (no prefix)
var defaultLogger = &TopicLogger{topic: ""}

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
	minLogLevel = l
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
func (t *TopicLogger) printf(format string, v ...any) {
	_, file, line, _ := runtime.Caller(3)
	logChannel <- logMessage{
		file:   file,
		line:   line,
		topic:  t.topic,
		format: format,
		args:   v,
	}
}

// Error writes an error log message with topic prefix
func (t *TopicLogger) Error(format string, v ...any) {
	if minLogLevel > ERROR {
		return
	}
	t.printf("ERROR: "+format, v...)
}

// Warn writes a warning log message with topic prefix
func (t *TopicLogger) Warn(format string, v ...any) {
	if minLogLevel > WARN {
		return
	}
	t.printf("WARN: "+format, v...)
}

// Fatal writes a fatal error log message with topic prefix and exits
func (t *TopicLogger) Fatal(format string, v ...any) {
	t.printf("FATAL: "+format, v...)
	os.Exit(1)
}

// Info writes an info log message with topic prefix
func (t *TopicLogger) Info(format string, v ...any) {
	if minLogLevel > INFO {
		return
	}
	t.printf("INFO: "+format, v...)
}

// Debug writes a debug log message with topic prefix
func (t *TopicLogger) Debug(format string, v ...any) {
	if minLogLevel > DEBUG {
		return
	}
	t.printf("DEBUG: "+format, v...)
}

// Package-minLogLevel convenience functions that delegate to defaultLogger

// Error writes an error log message
func Error(format string, v ...any) {
	defaultLogger.Error(format, v...)
}

// Warn writes a warning log message
func Warn(format string, v ...any) {
	defaultLogger.Warn(format, v...)
}

// Fatal writes a fatal error log message and exits
func Fatal(format string, v ...any) {
	defaultLogger.Fatal(format, v...)
}

// Info writes an info log message
func Info(format string, v ...any) {
	defaultLogger.Info(format, v...)
}

// Debug writes a debug log message
func Debug(format string, v ...any) {
	defaultLogger.Debug(format, v...)
}
