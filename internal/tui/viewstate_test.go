package tui

import (
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
	ctypes "git.15b.it/eno/critic/pkg/types"
)

func TestNewViewState(t *testing.T) {
	vs := NewViewState()
	assert.NotNil(t, vs, "NewViewState() returned nil")
	assert.Equals(t, vs.GetSelectedFile(), "", "Initial selected file should be empty")
	assert.Equals(t, vs.activeLine, -1, "Initial active line should be -1")
}

func TestViewState_SetSelectedFile(t *testing.T) {
	vs := NewViewState()

	vs.SetSelectedFile("test.go")

	assert.Equals(t, vs.GetSelectedFile(), "test.go", "GetSelectedFile()")

	// Active line should be reset when file changes
	vs.SetActiveLine(5)
	vs.SetSelectedFile("other.go")

	assert.Equals(t, vs.activeLine, -1, "Active line should be reset to -1 when file changes")
}

func TestViewState_SetActiveLine(t *testing.T) {
	vs := NewViewState()

	vs.SetActiveLine(10)

	assert.Equals(t, vs.activeLine, 10, "activeLine")
}

func TestViewState_GetActiveHunkPosition(t *testing.T) {
	vs := NewViewState()

	// Should return nil when no file is set
	assert.Nil(t, vs.GetActiveHunkPosition(), "GetActiveHunkPosition() should return nil when no file is set")

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
	assert.NotNil(t, pos, "GetActiveHunkPosition() returned nil")
	assert.Equals(t, pos.HunkIndex, 0, "HunkIndex")
	assert.Equals(t, pos.LineInHunk, 1, "LineInHunk")
	assert.Equals(t, pos.TotalLinesInHunk, 3, "TotalLinesInHunk")

	// Test second hunk
	vs.SetActiveLine(4) // Second line in second hunk (3 lines in first hunk + 1)

	pos = vs.GetActiveHunkPosition()
	assert.NotNil(t, pos, "GetActiveHunkPosition() returned nil for second hunk")
	assert.Equals(t, pos.HunkIndex, 1, "HunkIndex for second hunk")
	assert.Equals(t, pos.LineInHunk, 1, "LineInHunk for second hunk")
	assert.Equals(t, pos.TotalLinesInHunk, 2, "TotalLinesInHunk for second hunk")
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

	assert.Nil(t, vs.GetActiveHunkPosition(), "GetActiveHunkPosition() should return nil for out of range line")
}
