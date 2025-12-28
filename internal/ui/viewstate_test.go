package ui

import (
	"testing"

	ctypes "git.15b.it/eno/critic/pkg/types"
)

func TestNewViewState(t *testing.T) {
	vs := NewViewState()
	if vs == nil {
		t.Fatal("NewViewState() returned nil")
	}

	if vs.GetSelectedFile() != "" {
		t.Errorf("Initial selected file should be empty, got %q", vs.GetSelectedFile())
	}

	if vs.activeLine != -1 {
		t.Errorf("Initial active line should be -1, got %d", vs.activeLine)
	}
}

func TestViewState_SetSelectedFile(t *testing.T) {
	vs := NewViewState()

	vs.SetSelectedFile("test.go")

	if vs.GetSelectedFile() != "test.go" {
		t.Errorf("GetSelectedFile() = %q, want test.go", vs.GetSelectedFile())
	}

	// Active line should be reset when file changes
	vs.SetActiveLine(5)
	vs.SetSelectedFile("other.go")

	if vs.activeLine != -1 {
		t.Errorf("Active line should be reset to -1 when file changes, got %d", vs.activeLine)
	}
}

func TestViewState_SetActiveLine(t *testing.T) {
	vs := NewViewState()

	vs.SetActiveLine(10)

	if vs.activeLine != 10 {
		t.Errorf("activeLine = %d, want 10", vs.activeLine)
	}
}

func TestViewState_GetActiveHunkPosition(t *testing.T) {
	vs := NewViewState()

	// Should return nil when no file is set
	if pos := vs.GetActiveHunkPosition(); pos != nil {
		t.Error("GetActiveHunkPosition() should return nil when no file is set")
	}

	// Set up a file with hunks
	file := &ctypes.FileDiff{
		NewPath: "test.go",
		Hunks: []*ctypes.Hunk{
			{
				OldStart: 1,
				NewStart: 1,
				Lines: []*ctypes.Line{
					{Type: ctypes.LineContext, Content: "line 1"},
					{Type: ctypes.LineContext, Content: "line 2"},
					{Type: ctypes.LineAdded, Content: "line 3"},
				},
			},
			{
				OldStart: 10,
				NewStart: 10,
				Lines: []*ctypes.Line{
					{Type: ctypes.LineContext, Content: "line 10"},
					{Type: ctypes.LineDeleted, Content: "line 11"},
				},
			},
		},
	}

	vs.SetFile(file)
	vs.SetActiveLine(1) // Second line in first hunk

	pos := vs.GetActiveHunkPosition()
	if pos == nil {
		t.Fatal("GetActiveHunkPosition() returned nil")
	}

	if pos.HunkIndex != 0 {
		t.Errorf("HunkIndex = %d, want 0", pos.HunkIndex)
	}

	if pos.LineInHunk != 1 {
		t.Errorf("LineInHunk = %d, want 1", pos.LineInHunk)
	}

	if pos.TotalLinesInHunk != 3 {
		t.Errorf("TotalLinesInHunk = %d, want 3", pos.TotalLinesInHunk)
	}

	// Test second hunk
	vs.SetActiveLine(4) // Second line in second hunk (3 lines in first hunk + 1)

	pos = vs.GetActiveHunkPosition()
	if pos == nil {
		t.Fatal("GetActiveHunkPosition() returned nil for second hunk")
	}

	if pos.HunkIndex != 1 {
		t.Errorf("HunkIndex = %d, want 1", pos.HunkIndex)
	}

	if pos.LineInHunk != 1 {
		t.Errorf("LineInHunk = %d, want 1", pos.LineInHunk)
	}

	if pos.TotalLinesInHunk != 2 {
		t.Errorf("TotalLinesInHunk = %d, want 2", pos.TotalLinesInHunk)
	}
}

func TestViewState_GetActiveHunkPosition_OutOfRange(t *testing.T) {
	vs := NewViewState()

	file := &ctypes.FileDiff{
		NewPath: "test.go",
		Hunks: []*ctypes.Hunk{
			{
				Lines: []*ctypes.Line{
					{Type: ctypes.LineContext, Content: "line 1"},
				},
			},
		},
	}

	vs.SetFile(file)
	vs.SetActiveLine(999) // Out of range

	pos := vs.GetActiveHunkPosition()
	if pos != nil {
		t.Error("GetActiveHunkPosition() should return nil for out of range line")
	}
}
