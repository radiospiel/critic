package tui

import (
	ctypes "git.15b.it/eno/critic/src/pkg/types"
)

// HunkPosition represents the position within a hunk
type HunkPosition struct {
	HunkIndex     int         // Index of the hunk in the file
	Hunk          *ctypes.Hunk // The hunk itself
	LineInHunk    int         // Line number within the hunk (0-based)
	TotalLinesInHunk int      // Total number of lines in the hunk
}

// ViewState provides access to the current view state
type ViewState interface {
	// GetSelectedFile returns the name/path of the currently selected file
	// Returns empty string if no file is selected
	GetSelectedFile() string

	// GetActiveHunkPosition returns the hunk and position that the active line is in
	// Returns nil if no file is selected or no active line
	GetActiveHunkPosition() *HunkPosition

	// SetSelectedFile sets the currently selected file
	SetSelectedFile(path string)

	// SetActiveLine sets the active line number (used to determine which hunk)
	SetActiveLine(lineNum int)
}

// DefaultViewState is a concrete implementation of ViewState
type DefaultViewState struct {
	selectedFile string
	activeLine   int
	file         *ctypes.FileDiff
}

// NewViewState creates a new DefaultViewState
func NewViewState() *DefaultViewState {
	return &DefaultViewState{
		selectedFile: "",
		activeLine:   -1,
	}
}

// GetSelectedFile returns the currently selected file path
func (v *DefaultViewState) GetSelectedFile() string {
	return v.selectedFile
}

// GetActiveHunkPosition returns the hunk and position of the active line
func (v *DefaultViewState) GetActiveHunkPosition() *HunkPosition {
	if v.file == nil || v.activeLine < 0 {
		return nil
	}

	// Find which hunk contains the active line
	lineCount := 0
	for hunkIdx, hunk := range v.file.Hunks {
		hunkLines := len(hunk.Lines)
		if v.activeLine >= lineCount && v.activeLine < lineCount+hunkLines {
			return &HunkPosition{
				HunkIndex:        hunkIdx,
				Hunk:             hunk,
				LineInHunk:       v.activeLine - lineCount,
				TotalLinesInHunk: hunkLines,
			}
		}
		lineCount += hunkLines
	}

	return nil
}

// SetSelectedFile sets the currently selected file
func (v *DefaultViewState) SetSelectedFile(path string) {
	v.selectedFile = path
	v.activeLine = -1 // Reset active line when file changes
}

// SetActiveLine sets the active line number
func (v *DefaultViewState) SetActiveLine(lineNum int) {
	v.activeLine = lineNum
}

// SetFile sets the current file being viewed (needed for hunk calculations)
func (v *DefaultViewState) SetFile(file *ctypes.FileDiff) {
	v.file = file
}
