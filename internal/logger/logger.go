package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	logger *log.Logger
	once   sync.Once
)

// Init initializes the logger
func Init() error {
	var err error
	once.Do(func() {
		f, e := os.OpenFile("/tmp/critic.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if e != nil {
			err = e
			return
		}
		logger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
	})
	return err
}

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
	Log("ERROR: "+format, v...)
}

// Info writes an info log message
func Info(format string, v ...interface{}) {
	Log("INFO: "+format, v...)
}

// Debug writes a debug log message
func Debug(format string, v ...interface{}) {
	Log("DEBUG: "+format, v...)
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
		msg := fmt.Sprint(v...)
		logger.Print(msg)
	}
}
