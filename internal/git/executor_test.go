package git

import (
	"fmt"
	"testing"
)

// MockExecutor is a test double for CommandExecutor
type MockExecutor struct {
	// Commands stores all commands that were executed
	Commands [][]string
	// Output is the output to return for each call
	Output []byte
	// Err is the error to return
	Err error
}

func (m *MockExecutor) Run(name string, args ...string) ([]byte, error) {
	// Record the command
	cmd := append([]string{name}, args...)
	m.Commands = append(m.Commands, cmd)

	return m.Output, m.Err
}

func TestDefaultExecutor_Run(t *testing.T) {
	executor := &DefaultExecutor{}

	// Test a simple command that should work on any system
	output, err := executor.Run("echo", "hello")
	if err != nil {
		t.Fatalf("DefaultExecutor.Run() error = %v", err)
	}

	// Output should contain "hello"
	if len(output) == 0 {
		t.Error("DefaultExecutor.Run() returned empty output")
	}
}

func TestDefaultExecutor_RunError(t *testing.T) {
	executor := &DefaultExecutor{}

	// Test a command that should fail
	_, err := executor.Run("nonexistent-command-xyz-123")
	if err == nil {
		t.Error("DefaultExecutor.Run() should return error for nonexistent command")
	}
}

func TestMockExecutor_RecordsCommands(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte("test output"),
		Err:    nil,
	}

	// Execute a command
	output, err := mock.Run("git", "status")
	if err != nil {
		t.Fatalf("MockExecutor.Run() error = %v", err)
	}

	if string(output) != "test output" {
		t.Errorf("MockExecutor.Run() = %q, want %q", output, "test output")
	}

	// Verify command was recorded
	if len(mock.Commands) != 1 {
		t.Fatalf("MockExecutor recorded %d commands, want 1", len(mock.Commands))
	}

	cmd := mock.Commands[0]
	if len(cmd) != 2 || cmd[0] != "git" || cmd[1] != "status" {
		t.Errorf("MockExecutor recorded command %v, want [git status]", cmd)
	}
}

func TestMockExecutor_ReturnsError(t *testing.T) {
	expectedErr := fmt.Errorf("command failed")
	mock := &MockExecutor{
		Output: nil,
		Err:    expectedErr,
	}

	_, err := mock.Run("git", "fail")
	if err != expectedErr {
		t.Errorf("MockExecutor.Run() error = %v, want %v", err, expectedErr)
	}
}

func TestMockExecutor_MultipleCommands(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte("output"),
	}

	// Execute multiple commands
	mock.Run("git", "status")
	mock.Run("git", "diff")
	mock.Run("echo", "hello")

	if len(mock.Commands) != 3 {
		t.Fatalf("MockExecutor recorded %d commands, want 3", len(mock.Commands))
	}

	// Verify first command
	if mock.Commands[0][0] != "git" || mock.Commands[0][1] != "status" {
		t.Errorf("First command = %v, want [git status]", mock.Commands[0])
	}

	// Verify second command
	if mock.Commands[1][0] != "git" || mock.Commands[1][1] != "diff" {
		t.Errorf("Second command = %v, want [git diff]", mock.Commands[1])
	}

	// Verify third command
	if mock.Commands[2][0] != "echo" || mock.Commands[2][1] != "hello" {
		t.Errorf("Third command = %v, want [echo hello]", mock.Commands[2])
	}
}
