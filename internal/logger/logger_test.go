package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewFileLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Fatal("NewFileLogger() returned nil")
	}

	if logger.logger == nil {
		t.Error("FileLogger.logger is nil")
	}

	if logger.file == nil {
		t.Error("FileLogger.file is nil")
	}
}

func TestNewFileLogger_InvalidPath(t *testing.T) {
	_, err := NewFileLogger("/nonexistent/invalid/path/test.log")
	if err == nil {
		t.Error("NewFileLogger() should return error for invalid path")
	}
}

func TestFileLogger_Methods(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	l, err := NewFileLogger(logPath)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	defer l.Close()

	// Test all logging methods - they should not panic
	l.Info("test info %s", "message")
	l.Error("test error %s", "message")
	l.Debug("test debug %s", "message")
	l.Log("test log %s", "message")
	l.Print("test print")
	l.Println("test println")

	// Verify log file exists and has content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file is empty")
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "INFO: test info message") {
		t.Error("Log should contain INFO message")
	}
	if !strings.Contains(contentStr, "ERROR: test error message") {
		t.Error("Log should contain ERROR message")
	}
	if !strings.Contains(contentStr, "DEBUG: test debug message") {
		t.Error("Log should contain DEBUG message")
	}
}

func TestFileLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	l, err := NewFileLogger(logPath)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}

	err = l.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNullLogger(t *testing.T) {
	// NullLogger should not panic on any method
	l := &NullLogger{}

	l.Info("test")
	l.Error("test")
	l.Debug("test")
	l.Log("test")
	l.Print("test")
	l.Println("test")
}

func TestInit(t *testing.T) {
	// Reset the logger state for this test
	logger = nil
	once = sync.Once{}

	err := Init()
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	// Calling Init again should not reinitialize
	err = Init()
	if err != nil {
		t.Errorf("Second Init() error = %v", err)
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Reset and initialize logger
	logger = nil
	once = sync.Once{}

	err := Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test global functions - should not panic
	Log("test %s", "log")
	Logf("test %s", "logf")
	Printf("test %s", "printf")
	Error("test %s", "error")
	Info("test %s", "info")
	Debug("test %s", "debug")
	Print("test")
	Println("test")
}
