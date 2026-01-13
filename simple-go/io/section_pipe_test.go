package io

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewSectionPipe(t *testing.T) {
	pipe := NewSectionPipe(5, 10)
	if pipe.skip != 5 {
		t.Errorf("expected skip=5, got %d", pipe.skip)
	}
	if pipe.take != 10 {
		t.Errorf("expected take=10, got %d", pipe.take)
	}
}

func TestNewSectionPipeLines(t *testing.T) {
	// Lines 5-10 means skip 4, take 6
	pipe := NewSectionPipeLines(5, 10)
	if pipe.skip != 4 {
		t.Errorf("expected skip=4, got %d", pipe.skip)
	}
	if pipe.take != 6 {
		t.Errorf("expected take=6, got %d", pipe.take)
	}
}

func TestNewSectionPipeLines_StartLineLessThanOne(t *testing.T) {
	// Start line 0 should be treated as 1
	pipe := NewSectionPipeLines(0, 5)
	if pipe.skip != 0 {
		t.Errorf("expected skip=0, got %d", pipe.skip)
	}
	if pipe.take != 5 {
		t.Errorf("expected take=5, got %d", pipe.take)
	}
}

func TestNewSectionPipeLines_EndBeforeStart(t *testing.T) {
	// End before start should result in take=0
	pipe := NewSectionPipeLines(10, 5)
	if pipe.skip != 9 {
		t.Errorf("expected skip=9, got %d", pipe.skip)
	}
	if pipe.take != 0 {
		t.Errorf("expected take=0, got %d", pipe.take)
	}
}

func TestSectionPipe_BasicFiltering(t *testing.T) {
	// Create a pipe that skips 2 lines and takes 3 lines
	pipe := NewSectionPipe(2, 3)
	pr, pw := pipe.Pipe()

	// Write test data in a goroutine
	go func() {
		defer pw.Close()
		pw.Write([]byte("line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\n"))
	}()

	// Read the filtered output
	output, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "line 3\nline 4\nline 5\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestSectionPipe_Lines(t *testing.T) {
	// Create a pipe that extracts lines 3-5 (1-indexed, inclusive)
	pipe := NewSectionPipeLines(3, 5)
	pr, pw := pipe.Pipe()

	// Write test data
	go func() {
		defer pw.Close()
		pw.Write([]byte("line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\n"))
	}()

	output, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "line 3\nline 4\nline 5\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestSectionPipe_SkipAll(t *testing.T) {
	// Skip more lines than available
	pipe := NewSectionPipe(100, 5)
	pr, pw := pipe.Pipe()

	go func() {
		defer pw.Close()
		pw.Write([]byte("line 1\nline 2\nline 3\n"))
	}()

	output, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(output) != "" {
		t.Errorf("expected empty output, got %q", string(output))
	}
}

func TestSectionPipe_TakeAll(t *testing.T) {
	// Skip 0 and take a large number
	pipe := NewSectionPipe(0, 100)
	pr, pw := pipe.Pipe()

	go func() {
		defer pw.Close()
		pw.Write([]byte("line 1\nline 2\nline 3\n"))
	}()

	output, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "line 1\nline 2\nline 3\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestSectionPipe_TakeZero(t *testing.T) {
	// Take 0 lines
	pipe := NewSectionPipe(0, 0)
	pr, pw := pipe.Pipe()

	go func() {
		defer pw.Close()
		pw.Write([]byte("line 1\nline 2\nline 3\n"))
	}()

	output, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(output) != "" {
		t.Errorf("expected empty output, got %q", string(output))
	}
}

func TestSectionPipe_LargeInput(t *testing.T) {
	// Test with a large number of lines to ensure draining works
	pipe := NewSectionPipe(5, 3)
	pr, pw := pipe.Pipe()

	go func() {
		defer pw.Close()
		for i := 1; i <= 1000; i++ {
			pw.Write([]byte("line content\n"))
		}
	}()

	output, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have exactly 3 lines
	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestSectionPipe_MultipleWrites(t *testing.T) {
	// Test that multiple small writes work correctly
	pipe := NewSectionPipe(1, 2)
	pr, pw := pipe.Pipe()

	go func() {
		defer pw.Close()
		pw.Write([]byte("line 1\n"))
		pw.Write([]byte("line 2\n"))
		pw.Write([]byte("line 3\n"))
		pw.Write([]byte("line 4\n"))
	}()

	output, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "line 2\nline 3\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestSectionPipe_PartialLines(t *testing.T) {
	// Test that partial line writes are handled correctly
	pipe := NewSectionPipe(0, 2)
	pr, pw := pipe.Pipe()

	go func() {
		defer pw.Close()
		pw.Write([]byte("line"))
		pw.Write([]byte(" 1\nli"))
		pw.Write([]byte("ne 2\nline 3\n"))
	}()

	output, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "line 1\nline 2\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

func TestSectionPipe_EmptyInput(t *testing.T) {
	pipe := NewSectionPipe(0, 5)
	pr, pw := pipe.Pipe()

	go func() {
		defer pw.Close()
		// Write nothing
	}()

	output, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(output) != "" {
		t.Errorf("expected empty output, got %q", string(output))
	}
}

func TestSectionPipe_NoTrailingNewline(t *testing.T) {
	// Input without trailing newline - the last line should still be captured
	pipe := NewSectionPipe(0, 3)
	pr, pw := pipe.Pipe()

	go func() {
		defer pw.Close()
		pw.Write([]byte("line 1\nline 2\nline 3"))
	}()

	output, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// bufio.Scanner captures the last line even without a trailing newline
	// Our implementation adds newlines to all lines
	expected := "line 1\nline 2\nline 3\n"
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}
}

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
	output, err := ReadFileLines(tmpfile.Name(), 3, 5)
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
	output, err := ReadFileLines(tmpfile.Name(), 0, 2)
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
	output, err := ReadFileLines(tmpfile.Name(), 2, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "line 2\nline 3\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestReadFileLines_FileNotFound(t *testing.T) {
	_, err := ReadFileLines("/nonexistent/file/path", 1, 10)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
