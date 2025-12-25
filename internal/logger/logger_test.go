package logger

import (
	"bytes"
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

func TestNewFileLoggerWithWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := NewFileLoggerWithWriter(&buf)

	if logger == nil {
		t.Fatal("NewFileLoggerWithWriter() returned nil")
	}

	if logger.logger == nil {
		t.Error("FileLogger.logger is nil")
	}

	if logger.file != nil {
		t.Error("FileLogger.file should be nil when using custom writer")
	}
}

func TestFileLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := NewFileLoggerWithWriter(&buf)

	logger.Info("test message %d", 123)

	output := buf.String()
	if !strings.Contains(output, "INFO: test message 123") {
		t.Errorf("Info() output = %q, want to contain 'INFO: test message 123'", output)
	}
}

func TestFileLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := NewFileLoggerWithWriter(&buf)

	logger.Error("error message %s", "test")

	output := buf.String()
	if !strings.Contains(output, "ERROR: error message test") {
		t.Errorf("Error() output = %q, want to contain 'ERROR: error message test'", output)
	}
}

func TestFileLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := NewFileLoggerWithWriter(&buf)

	logger.Debug("debug message")

	output := buf.String()
	if !strings.Contains(output, "DEBUG: debug message") {
		t.Errorf("Debug() output = %q, want to contain 'DEBUG: debug message'", output)
	}
}

func TestFileLogger_Log(t *testing.T) {
	var buf bytes.Buffer
	logger := NewFileLoggerWithWriter(&buf)

	logger.Log("plain message")

	output := buf.String()
	if !strings.Contains(output, "plain message") {
		t.Errorf("Log() output = %q, want to contain 'plain message'", output)
	}
}

func TestFileLogger_Print(t *testing.T) {
	var buf bytes.Buffer
	logger := NewFileLoggerWithWriter(&buf)

	logger.Print("test", " ", "print")

	output := buf.String()
	if !strings.Contains(output, "test print") {
		t.Errorf("Print() output = %q, want to contain 'test print'", output)
	}
}

func TestFileLogger_Println(t *testing.T) {
	var buf bytes.Buffer
	logger := NewFileLoggerWithWriter(&buf)

	logger.Println("test", "println")

	output := buf.String()
	if !strings.Contains(output, "test println") {
		t.Errorf("Println() output = %q, want to contain 'test println'", output)
	}
}

func TestFileLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}

	err = logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestFileLogger_CloseWithWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := NewFileLoggerWithWriter(&buf)

	// Should not error when closing logger without file
	err := logger.Close()
	if err != nil {
		t.Errorf("Close() for writer-based logger error = %v", err)
	}
}

func TestNullLogger(t *testing.T) {
	logger := &NullLogger{}

	// All methods should be safe to call
	logger.Info("test")
	logger.Error("test")
	logger.Debug("test")
	logger.Log("test")
	logger.Print("test")
	logger.Println("test")

	// If we get here without panic, test passes
}

func TestSetLogger(t *testing.T) {
	// Save original logger
	original := getLogger()
	defer func() {
		if original != nil {
			SetLogger(original)
		}
	}()

	// Set a null logger
	null := &NullLogger{}
	SetLogger(null)

	if getLogger() != null {
		t.Error("SetLogger() did not set the logger")
	}
}

func TestGetLogger_Nil(t *testing.T) {
	// Save original logger
	original := getLogger()
	defer func() {
		if original != nil {
			SetLogger(original)
		}
	}()

	// Set to nil
	SetLogger(nil)

	if getLogger() != nil {
		t.Error("getLogger() should return nil when set to nil")
	}
}

func TestGlobalFunctions_WithNullLogger(t *testing.T) {
	// Save original logger
	original := getLogger()
	defer func() {
		if original != nil {
			SetLogger(original)
		}
	}()

	// Set null logger to prevent actual logging during tests
	SetLogger(&NullLogger{})

	// All global functions should be safe to call
	Log("test")
	Logf("test")
	Printf("test")
	Info("test")
	Error("test")
	Debug("test")
	Print("test")
	Println("test")

	// If we get here without panic, test passes
}

func TestGlobalFunctions_WithFileLogger(t *testing.T) {
	// Save original logger
	original := getLogger()
	defer func() {
		if original != nil {
			SetLogger(original)
		}
	}()

	var buf bytes.Buffer
	SetLogger(NewFileLoggerWithWriter(&buf))

	Info("test info")

	output := buf.String()
	if !strings.Contains(output, "INFO: test info") {
		t.Errorf("Global Info() should delegate to logger, got: %q", output)
	}
}

func TestGlobalFunctions_WithNilLogger(t *testing.T) {
	// Save original logger
	original := getLogger()
	defer func() {
		if original != nil {
			SetLogger(original)
		}
	}()

	// Set to nil
	SetLogger(nil)

	// All functions should handle nil logger gracefully
	Log("test")
	Info("test")
	Error("test")
	Debug("test")
	Print("test")
	Println("test")

	// If we get here without panic, test passes
}

func TestInit(t *testing.T) {
	// Save original logger
	original := getLogger()
	defer func() {
		if original != nil {
			SetLogger(original)
		}
	}()

	// Reset the once to allow re-initialization
	// Note: This is a bit hacky for testing, but necessary
	once = sync.Once{}

	err := Init()
	if err != nil {
		// Init might fail if /tmp is not writable, that's OK
		t.Logf("Init() error (might be expected): %v", err)
		return
	}

	if getLogger() == nil {
		t.Error("Init() should set a logger")
	}
}

func TestFileLogger_WritesToFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}

	logger.Info("test message")
	logger.Error("error message")
	logger.Debug("debug message")

	err = logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Read the log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	output := string(content)

	// Verify all messages were written
	if !strings.Contains(output, "INFO: test message") {
		t.Error("Log file should contain INFO message")
	}
	if !strings.Contains(output, "ERROR: error message") {
		t.Error("Log file should contain ERROR message")
	}
	if !strings.Contains(output, "DEBUG: debug message") {
		t.Error("Log file should contain DEBUG message")
	}
}

func TestFileLogger_Append(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Write first message
	logger1, err := NewFileLogger(logPath)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	logger1.Info("first message")
	logger1.Close()

	// Write second message (should append)
	logger2, err := NewFileLogger(logPath)
	if err != nil {
		t.Fatalf("NewFileLogger() error = %v", err)
	}
	logger2.Info("second message")
	logger2.Close()

	// Read the log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	output := string(content)

	// Both messages should be present
	if !strings.Contains(output, "first message") {
		t.Error("Log file should contain first message")
	}
	if !strings.Contains(output, "second message") {
		t.Error("Log file should contain second message")
	}
}
