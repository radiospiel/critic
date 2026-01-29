package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// SimpleLogger is a logger with support for topics, and filenames
type SimpleLogger struct {
	topic string
	file  string
	line  int
	level Level
}

var defaultLogger = SimpleLogger{
	topic: "",
	file:  "",
	line:  0,
	level: INFO,
}

// WithCaller returns a logger that uses the provided caller info
func (sl *SimpleLogger) deepCopy(fun func(copy *SimpleLogger)) *SimpleLogger {
	copy := &SimpleLogger{
		topic: sl.topic,
		file:  sl.file,
		line:  sl.line,
		level: sl.level,
	}

	fun(copy)
	return copy
}

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
				if strings.HasPrefix(file, wd+"/") {
					file = file[len(wd)+1:]
				}
				msg = fmt.Sprintf("%s(%d): ", file, entry.line) + msg
			}
			fileLogger.Println(msg)
		}
	}()
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
	defaultLogger.level = l
}

// WithTopic returns a SimpleLogger that prepends [topic] to all log messages
func WithTopic(topic string) *SimpleLogger {
	return defaultLogger.WithTopic(topic)
}

// WithCaller returns a logger that uses the provided caller info
func WithCaller(file string, line int) *SimpleLogger {
	return defaultLogger.WithCaller(file, line)
}

// WithCaller returns a logger that uses the provided caller info
func (sl *SimpleLogger) WithLevel(level Level) *SimpleLogger {
	return sl.deepCopy(func(copy *SimpleLogger) {
		copy.level = level
	})
}

// WithCaller returns a logger that uses the provided caller info
func (sl *SimpleLogger) WithCaller(file string, line int) *SimpleLogger {
	return sl.deepCopy(func(copy *SimpleLogger) {
		copy.file = file
		copy.line = line
	})
}

// WithCaller returns a logger that uses the provided caller info
func (sl *SimpleLogger) WithTopic(topic string) *SimpleLogger {
	return sl.deepCopy(func(copy *SimpleLogger) {
		copy.topic = topic
	})
}

// printf writes a log message with optional topic prefix
func (t *SimpleLogger) printf(format string, v ...any) {
	file, line := t.file, t.line
	if file == "" {
		_, file, line, _ = runtime.Caller(3)
	}
	logChannel <- logMessage{
		file:   file,
		line:   line,
		topic:  t.topic,
		format: format,
		args:   v,
	}
}

// Fatal writes a fatal error log message with topic prefix and exits
func (t *SimpleLogger) Fatal(format string, v ...any) {
	t.printf("FATAL: "+format, v...)
	os.Exit(1)
}

// Error writes an error log message with topic prefix
func (t *SimpleLogger) Error(format string, v ...any) {
	if t.level > ERROR {
		return
	}
	t.printf("ERROR: "+format, v...)
}

// Warn writes a warning log message with topic prefix
func (t *SimpleLogger) Warn(format string, v ...any) {
	if t.level > WARN {
		return
	}
	t.printf("WARN: "+format, v...)
}

// Info writes an info log message with topic prefix
func (t *SimpleLogger) Info(format string, v ...any) {
	if t.level > INFO {
		return
	}
	t.printf("INFO: "+format, v...)
}

// Debug writes a debug log message with topic prefix
func (t *SimpleLogger) Debug(format string, v ...any) {
	if t.level > DEBUG {
		return
	}
	t.printf("DEBUG: "+format, v...)
}

// Fatal writes a fatal error log message and exits
func Fatal(format string, v ...any) {
	defaultLogger.Fatal(format, v...)
}

// Error writes an error log message
func Error(format string, v ...any) {
	defaultLogger.Error(format, v...)
}

// Warn writes a warning log message
func Warn(format string, v ...any) {
	defaultLogger.Warn(format, v...)
}

// Info writes an info log message
func Info(format string, v ...any) {
	defaultLogger.Info(format, v...)
}

// Debug writes a debug log message
func Debug(format string, v ...any) {
	defaultLogger.Debug(format, v...)
}
