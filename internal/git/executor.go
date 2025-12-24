package git

import "os/exec"

// CommandExecutor is an interface for executing shell commands.
// This allows us to mock command execution in tests.
type CommandExecutor interface {
	// Run executes a command and returns its output
	Run(name string, args ...string) ([]byte, error)
}

// DefaultExecutor is the standard executor that runs real commands
type DefaultExecutor struct{}

// Run executes a command using os/exec
func (e *DefaultExecutor) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// defaultExecutor is the package-level default executor
var defaultExecutor CommandExecutor = &DefaultExecutor{}
