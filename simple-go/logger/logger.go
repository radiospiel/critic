package logger

import (
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

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
var logFilePath = findLogFileName()

func findLogFileName() string {
	logFile := os.Getenv("LOG_FILE")
	if logFile != "" {
		return logFile
	}
	return "/dev/stderr"
}

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

// done signals the background goroutine to exit
var done = make(chan struct{})

var wd = (func() string { dir, _ := os.Getwd(); return dir })()

// transformArgs converts any Inspect-implementing values to their string representation
func transformArgs(args []any) []any {
	result := make([]any, len(args))
	for i, arg := range args {
		if protoMsg, ok := arg.(proto.Message); ok {
			// Use canonical protojson for protobuf messages
			arg = protojson.Format(protoMsg)
		}
		if argStr, ok := arg.(string); ok {
			// Truncate string arguments
			arg = truncate(argStr, maxLogLength)
		}

		result[i] = arg
	}
	return result
}

const maxLogLength = 200

// truncate cuts the string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// processLogEntry formats and writes a log entry
func processLogEntry(entry logMessage) {
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

func init() {
	// Start background writer goroutine
	go func() {
		for {
			select {
			case entry, ok := <-logChannel:
				if !ok {
					return
				}
				processLogEntry(entry)
			case <-done:
				// Drain remaining messages before exiting
				for {
					select {
					case entry := <-logChannel:
						processLogEntry(entry)
					default:
						return
					}
				}
			}
		}
	}()
}

// Close shuts down the logger goroutine. Should be called before program exit
// if you want to ensure all log messages are flushed.
func Close() {
	close(done)
}

func Runtime[T any](msg string, fun func() T) T {
	start := time.Now()
	r := fun()

	durationMs := float64(time.Since(start).Microseconds()) / 1000.0
	Info("%s: %.1f msecs", msg, durationMs)
	return r
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
func (t *SimpleLogger) Error(format string, v ...any) bool {
	if t.level > ERROR {
		return false
	}
	t.printf("ERROR: "+format, v...)
	return true
}

// Warn writes a warning log message with topic prefix
func (t *SimpleLogger) Warn(format string, v ...any) bool {
	if t.level > WARN {
		return false
	}
	t.printf("WARN: "+format, v...)
	return true
}

// Info writes an info log message with topic prefix
func (t *SimpleLogger) Info(format string, v ...any) bool {
	if t.level > INFO {
		return false
	}
	t.printf("INFO: "+format, v...)
	return true
}

// Debug writes a debug log message with topic prefix
func (t *SimpleLogger) Debug(format string, v ...any) bool {
	if t.level > DEBUG {
		return false
	}
	t.printf("DEBUG: "+format, v...)
	return true
}

// Fatal writes a fatal error log message and exits
func Fatal(format string, v ...any) {
	defaultLogger.Fatal(format, v...)
}

// Error writes an error log message
func Error(format string, v ...any) bool {
	return defaultLogger.Error(format, v...)
}

// Warn writes a warning log message
func Warn(format string, v ...any) bool {
	return defaultLogger.Warn(format, v...)
}

// Info writes an info log message
func Info(format string, v ...any) bool {
	return defaultLogger.Info(format, v...)
}

// Debug writes a debug log message
func Debug(format string, v ...any) bool {
	return defaultLogger.Debug(format, v...)
}
