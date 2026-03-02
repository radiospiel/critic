package logger

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// LevelT represents a log level
type LevelT int

const (
	DEBUG LevelT = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = [...]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

func (l LevelT) String() string {
	return levelNames[l]
}

var levelColors = [...]string{
	DEBUG: "\033[36m", // cyan
	INFO:  "",
	WARN:  "\033[33m", // yellow
	ERROR: "\033[31m", // red
	FATAL: "\033[35m", // magenta
}

// SetLevelColor overrides the ANSI color for a given log level.
func SetLevelColor(level LevelT, color string) {
	levelColors[level] = color
}

const colorReset = "\033[0m"

// OSC 8 hyperlink escape sequences for clickable links in supported terminals.
const (
	oscOpen  = "\033]8;;"
	oscClose = "\033\\"
)

// urlPattern matches http:// and https:// URLs in log messages.
var urlPattern = regexp.MustCompile(`https?://[^\s"'` + "`" + `\x00-\x1f]+`)

// Hyperlink wraps text as a clickable OSC 8 hyperlink pointing to url.
// Only produces escape sequences when the log destination is a TTY.
func Hyperlink(url, text string) string {
	if !sharedDest.enableColors {
		return text
	}
	return oscOpen + url + oscClose + text + oscOpen + oscClose
}

// linkifyURLs replaces bare URLs in s with OSC 8 clickable hyperlinks.
func linkifyURLs(s string) string {
	return urlPattern.ReplaceAllStringFunc(s, func(url string) string {
		return oscOpen + url + oscClose + url + oscOpen + oscClose
	})
}

// SimpleLogger is a logger with support for topics, and filenames
// Note that most calls will call directly to the sharedInstance, but a caller
// might build a customized logger via logger.WithPrefix("prefix"). ... and
// that returns a temporary instance

type logDestination struct {
	mu           sync.Mutex
	logger       *log.Logger
	enableColors bool
}

type SimpleLogger struct {
	dest  *logDestination
	topic string
	file  string // file set explicitely via WithCaller
	line  int    // line set explicitely via WithCaller
	level LevelT
}

var sharedDest = &logDestination{}

var sharedInstance = SimpleLogger{
	dest:  sharedDest,
	level: INFO,
}

func Level() LevelT { return sharedInstance.level }

// WithCaller returns a logger that uses the provided caller info
func (sl *SimpleLogger) deepCopy(fun func(copy *SimpleLogger)) *SimpleLogger {
	copy := &SimpleLogger{
		dest:  sl.dest,
		topic: sl.topic,
		file:  sl.file,
		line:  sl.line,
		level: sl.level,
	}

	fun(copy)
	return copy
}

func init() {
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = "/dev/stderr"
	}
	SetLogFile(logFile)
}

// logMessage holds the data for a log entry
type logMessage struct {
	level  LevelT
	file   string
	line   int
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
		msg = fmt.Sprintf("%s:%d ", file, entry.line) + msg
	}
	msg = entry.level.String() + ": " + msg

	// apply color and hyperlinks
	if sharedDest.enableColors {
		msg = linkifyURLs(msg)
		color := levelColors[entry.level]
		if color != "" {
			msg = color + msg + colorReset
		}
	}
	sharedDest.logger.Println(msg)
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

// SetLogFile opens the given path and configures the package-level fileLogger
func SetLogFile(path string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("OpenFile(%s) failed: %s", path, err))
	}

	sharedDest.mu.Lock()
	defer sharedDest.mu.Unlock()
	sharedDest.enableColors = term.IsTerminal(int(f.Fd()))
	sharedDest.logger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
}

// SetLogFlags sets the flags on the underlying log.Logger (e.g. 0 to disable timestamps).
func SetLogFlags(flags int) {
	sharedDest.mu.Lock()
	defer sharedDest.mu.Unlock()
	if sharedDest.logger != nil {
		sharedDest.logger.SetFlags(flags)
	}
}

// SetNullLog sets the logger to discard all output
func SetNullLog() {
	SetLogFile("/dev/null")
}

// SetLevel sets the minimum log level
func SetLevel(l LevelT) {
	sharedInstance.level = l
}

// WithLevel returns a SimpleLogger that uses the provided log level
func WithLevel(level LevelT) *SimpleLogger {
	return sharedInstance.WithLevel(level)
}

// WithTopic returns a SimpleLogger that prepends [topic] to all log messages
func WithTopic(topic string) *SimpleLogger {
	return sharedInstance.WithTopic(topic)
}

// WithCaller returns a logger that uses the provided caller info
func WithCaller(file string, line int) *SimpleLogger {
	return sharedInstance.WithCaller(file, line)
}


// WithLevel returns a logger that uses the provided log level. This method allows chaining of WithXXX() calls.
func (sl *SimpleLogger) WithLevel(level LevelT) *SimpleLogger {
	return sl.deepCopy(func(copy *SimpleLogger) {
		copy.level = level
	})
}

// WithCaller returns a logger that uses the provided caller info. This method allows chaining of WithXXX() calls.
func (sl *SimpleLogger) WithCaller(file string, line int) *SimpleLogger {
	return sl.deepCopy(func(copy *SimpleLogger) {
		copy.file = file
		copy.line = line
	})
}

// WithTopic returns a SimpleLogger that prepends [topic] to all log messages. This method allows chaining of WithXXX() calls.
func (sl *SimpleLogger) WithTopic(topic string) *SimpleLogger {
	return sl.deepCopy(func(copy *SimpleLogger) {
		copy.topic = topic
	})
}

// printf writes a log message with optional topic prefix
func (t *SimpleLogger) printf(level LevelT, format string, v ...any) {
	// determine caller unless set explicitly
	file, line := t.file, t.line
	if file == "" {
		_, file, line, _ = runtime.Caller(3)
	}
	logChannel <- logMessage{
		level:  level,
		file:   file,
		line:   line,
		topic:  t.topic,
		format: format,
		args:   v,
	}
}

// Fatal writes a fatal error log message with topic prefix and exits
func (t *SimpleLogger) Fatal(format string, v ...any) {
	t.printf(FATAL, format, v...)
	os.Exit(1)
}

// Error writes an error log message with topic prefix
func (t *SimpleLogger) Error(format string, v ...any) bool {
	if t.level > ERROR {
		return false
	}
	t.printf(ERROR, format, v...)
	return true
}

// Warn writes a warning log message with topic prefix
func (t *SimpleLogger) Warn(format string, v ...any) bool {
	if t.level > WARN {
		return false
	}
	t.printf(WARN, format, v...)
	return true
}

// Info writes an info log message with topic prefix
func (t *SimpleLogger) Info(format string, v ...any) bool {
	if t.level > INFO {
		return false
	}
	t.printf(INFO, format, v...)
	return true
}

// Debug writes a debug log message with topic prefix
func (t *SimpleLogger) Debug(format string, v ...any) bool {
	if t.level > DEBUG {
		return false
	}
	t.printf(DEBUG, format, v...)
	return true
}

func Fatal(format string, v ...any) {
	sharedInstance.Fatal(format, v...)
}

func Error(format string, v ...any) bool {
	return sharedInstance.Error(format, v...)
}

func Warn(format string, v ...any) bool {
	return sharedInstance.Warn(format, v...)
}

func Info(format string, v ...any) bool {
	return sharedInstance.Info(format, v...)
}

func Debug(format string, v ...any) bool {
	return sharedInstance.Debug(format, v...)
}
