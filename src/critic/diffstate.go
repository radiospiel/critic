package critic

import (
	ctypes "git.15b.it/eno/critic/src/pkg/types"
)

// FileState represents the state of a file in the diff
type FileState int

const (
	FileCreated FileState = iota
	FileDeleted
	FileChanged
)

// String returns the string representation of the file state
func (fs FileState) String() string {
	switch fs {
	case FileCreated:
		return "created"
	case FileDeleted:
		return "deleted"
	case FileChanged:
		return "changed"
	default:
		return "unknown"
	}
}

// FileInfo represents basic information about a changed file
type FileInfo struct {
	Path  string
	State FileState
}

// DiffDetails contains detailed diff information for a file
type DiffDetails struct {
	Path            string
	Hunks           []*ctypes.Hunk
	OriginalContent string // Full content of original file
	CurrentContent  string // Full content of current file
}

// OnChangeCallback is called when diff details change
// oldDetails is nil if the file is new, newDetails is nil if the file is deleted or reverted
type OnChangeCallback func(oldDetails, newDetails *DiffDetails)

// DiffState provides access to the current diff state
type DiffState interface {
	// GetFiles returns a list of all changed files with their states
	GetFiles() []FileInfo

	// GetDiffDetails returns detailed diff information for a specific file
	GetDiffDetails(path string) (*DiffDetails, error)

	// OnChange registers a callback to be notified when diff details change
	// Returns a function that can be called to unregister the callback
	OnChange(callback OnChangeCallback) func()

	// Refresh updates the diff state (re-reads from source)
	Refresh() error
}
