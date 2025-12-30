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

	logger := NewFileLogger(logPath)

	if logger == nil {
		t.Fatal("NewFileLogger() returned nil")
	}

	// Test logging - should not panic
	logger.Printf("test message")

	// Verify log file exists and has content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file is empty")
	}

	if !strings.Contains(string(content), "test message") {
		t.Error("Log should contain test message")
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

	Init()

	// Calling Init again should not reinitialize
	Init()
}

func TestGlobalFunctions(t *testing.T) {
	// Reset and initialize logger
	logger = nil
	once = sync.Once{}

	Init()

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
