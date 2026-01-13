package must

import (
	"strings"
	"testing"

	sio "git.15b.it/eno/critic/simple-go/io"
)

func TestPipeInto_BasicFiltering(t *testing.T) {
	// Use printf to output multiple lines, then filter
	pipe := sio.NewSectionPipe(2, 3)
	output := PipeInto(pipe, "printf", "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\n")

	expected := "line 3\nline 4\nline 5\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestPipeInto_Lines(t *testing.T) {
	// Extract lines 2-4 using line-based API
	pipe := sio.NewSectionPipeLines(2, 4)
	output := PipeInto(pipe, "printf", "line 1\nline 2\nline 3\nline 4\nline 5\n")

	expected := "line 2\nline 3\nline 4\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestPipeInto_SkipAll(t *testing.T) {
	pipe := sio.NewSectionPipe(100, 5)
	output := PipeInto(pipe, "printf", "line 1\nline 2\nline 3\n")

	if string(output) != "" {
		t.Errorf("expected empty output, got %q", string(output))
	}
}

func TestPipeInto_TakeAll(t *testing.T) {
	pipe := sio.NewSectionPipe(0, 100)
	output := PipeInto(pipe, "printf", "line 1\nline 2\nline 3\n")

	expected := "line 1\nline 2\nline 3\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestPipeInto_LargeOutput(t *testing.T) {
	// Generate a large number of lines to test parallel execution
	// This would block if not done in parallel due to pipe buffer limits
	pipe := sio.NewSectionPipe(100, 10)

	// Use seq to generate many lines
	output := PipeInto(pipe, "seq", "1", "10000")

	// Should get lines 101-110
	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d", len(lines))
	}
	if lines[0] != "101" {
		t.Errorf("expected first line to be '101', got %q", lines[0])
	}
	if lines[9] != "110" {
		t.Errorf("expected last line to be '110', got %q", lines[9])
	}
}

func TestPipeInto_FirstLine(t *testing.T) {
	// Get just the first line
	pipe := sio.NewSectionPipeLines(1, 1)
	output := PipeInto(pipe, "printf", "first\nsecond\nthird\n")

	expected := "first\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestPipeInto_CommandWithArgs(t *testing.T) {
	// Test with a command that takes arguments
	pipe := sio.NewSectionPipeLines(1, 3)
	output := PipeInto(pipe, "seq", "10", "20")

	// Should get lines 10, 11, 12
	expected := "10\n11\n12\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestPipeInto_EmptyOutput(t *testing.T) {
	pipe := sio.NewSectionPipe(0, 10)
	output := PipeInto(pipe, "printf", "")

	if string(output) != "" {
		t.Errorf("expected empty output, got %q", string(output))
	}
}

func TestPipeInto_Panics_OnCommandFailure(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic on command failure")
		}
	}()

	pipe := sio.NewSectionPipe(0, 10)
	// This command should fail (non-existent command)
	PipeInto(pipe, "nonexistent_command_12345")
}
