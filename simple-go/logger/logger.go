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
var logger = newFileLogger(logFilePath)

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
	logger = newFileLogger(path)
}

// SetNullLog sets the logger to discard all output
func SetNullLog() {
	logger = newNullLogger()
}

// SetLevel sets the minimum log level
func SetLevel(l Level) {
	level = l
}

// Package-level convenience functions

// Printf writes a log message (compatibility)
func Printf(format string, v ...interface{}) {
	go func() {
		logger.Printf(format, v...)
		if enableStdoutLog {
			if prefix != "" {
				v = append([]interface{}{prefix}, v...)
				fmt.Printf("%s "+format+"\n", v...)
			} else {
				fmt.Printf(format+"\n", v...)
			}
		}
	}()
}

// Error writes an error log message
func Error(format string, v ...interface{}) {
	if level > ERROR {
		return
	}
	Printf("ERROR: "+format, v...)
}

// Warn writes a warning log message
func Warn(format string, v ...interface{}) {
	if level > WARN {
		return
	}
	Printf("WARN: "+format, v...)
}

// Fatal writes a fatal error log message and exits
func Fatal(format string, v ...interface{}) {
	Printf("FATAL: "+format, v...)
	os.Exit(1)
}

// Info writes an info log message
func Info(format string, v ...interface{}) {
	if level > INFO {
		return
	}
	Printf("INFO: "+format, v...)
}

// Debug writes a debug log message
func Debug(format string, v ...interface{}) {
	if level > DEBUG {
		return
	}
	Printf("DEBUG: "+format, v...)
}
