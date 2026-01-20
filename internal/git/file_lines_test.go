package git

import (
	"os"
	"testing"
)

func TestReadFileLines(t *testing.T) {
	// Create a temp file with test content
	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\n"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	// Test reading middle lines
	output, err := readFileLines(tmpfile.Name(), 3, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "line 3\nline 4\nline 5\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestReadFileLines_StartLineLessThanOne(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := "line 1\nline 2\nline 3\n"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	// Start line 0 should be treated as 1
	output, err := readFileLines(tmpfile.Name(), 0, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "line 1\nline 2\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestReadFileLines_EndBeyondFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := "line 1\nline 2\nline 3\n"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	// End line beyond file should return all available lines
	output, err := readFileLines(tmpfile.Name(), 2, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "line 2\nline 3\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestReadFileLines_FileNotFound(t *testing.T) {
	_, err := readFileLines("/nonexistent/file/path", 1, 10)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
