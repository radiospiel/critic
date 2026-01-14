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
	// Extract lines 2-4 (skip 1, take 3)
	pipe := sio.NewSectionPipe(1, 3)
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
	// Get just the first line (skip 0, take 1)
	pipe := sio.NewSectionPipe(0, 1)
	output := PipeInto(pipe, "printf", "first\nsecond\nthird\n")

	expected := "first\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestPipeInto_CommandWithArgs(t *testing.T) {
	// Test with a command that takes arguments (skip 0, take 3)
	pipe := sio.NewSectionPipe(0, 3)
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

func TestPipeInto_LargeBinaryData(t *testing.T) {
	// Generate ~1MB of base64 encoded random data and pipe through SectionPipe
	// This tests that the parallel exec/read works correctly with large binary data
	// and doesn't block due to pipe buffer limits.
	//
	// We use fold -w 76 to ensure consistent line wrapping across platforms
	// (macOS base64 doesn't wrap by default, GNU base64 wraps at 76 chars).
	// 76 chars per line + newline = 77 bytes per line
	// 1024768 bytes / 77 ≈ 13308 lines
	pipe := sio.NewSectionPipe(100, 1000)

	// Generate large binary data with explicit line wrapping via fold
	output := PipeInto(pipe, "bash", "-c", "base64 /dev/urandom | tr -d '\\n' | fold -w 76 | head -n 15000")

	// Should have exactly 1000 lines
	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(lines) != 1000 {
		t.Errorf("expected 1000 lines, got %d", len(lines))
	}

	// Each line should be base64 encoded (76 chars typically, last line may be shorter)
	for i, line := range lines[:10] { // Check first 10 lines
		if len(line) == 0 {
			t.Errorf("line %d is empty", i)
		}
		// base64 lines are typically 76 chars
		if len(line) > 77 {
			t.Errorf("line %d unexpectedly long: %d chars", i, len(line))
		}
	}
}
