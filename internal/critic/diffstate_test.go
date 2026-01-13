package critic

import (
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
	ctypes "git.15b.it/eno/critic/pkg/types"
)

func TestFileState_String(t *testing.T) {
	tests := []struct {
		state FileState
		want  string
	}{
		{FileCreated, "created"},
		{FileDeleted, "deleted"},
		{FileChanged, "changed"},
		{FileState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equals(t, tt.state.String(), tt.want)
		})
	}
}

func TestGitDiffState_GetFiles(t *testing.T) {
	// This test would require a git repository with changes
	// For now, just test the basic structure works
	state := &GitDiffState{
		diff: &ctypes.Diff{
			Files: []*ctypes.FileDiff{
				{
					NewPath: "test.go",
					IsNew:   true,
				},
				{
					OldPath:   "deleted.go",
					NewPath:   "deleted.go",
					IsDeleted: true,
				},
				{
					NewPath: "modified.go",
					OldPath: "modified.go",
				},
			},
		},
	}

	files := state.GetFiles()
	assert.Equals(t, len(files), 3, "expected 3 files")

	// Check first file (created)
	assert.Equals(t, files[0].Path, "test.go")
	assert.Equals(t, files[0].State, FileCreated)

	// Check second file (deleted)
	assert.Equals(t, files[1].Path, "deleted.go")
	assert.Equals(t, files[1].State, FileDeleted)

	// Check third file (changed)
	assert.Equals(t, files[2].Path, "modified.go")
	assert.Equals(t, files[2].State, FileChanged)
}

func TestGitDiffState_OnChange(t *testing.T) {
	state := &GitDiffState{
		diff:           &ctypes.Diff{},
		callbacks:      make(map[int]OnChangeCallback),
		nextCallbackID: 0,
	}

	called := false
	callback := func(old, new *DiffDetails) {
		called = true
	}

	// Register callback
	unregister := state.OnChange(callback)
	assert.Equals(t, len(state.callbacks), 1, "Expected 1 callback")

	// Unregister callback
	unregister()
	assert.Equals(t, len(state.callbacks), 0, "Expected 0 callbacks after unregister")

	// Callback should not have been called yet
	assert.False(t, called, "Callback was called unexpectedly")
}

func TestFilesEqual(t *testing.T) {
	file1 := &ctypes.FileDiff{
		NewPath: "test.go",
		OldPath: "test.go",
		IsNew:   false,
		Hunks: []*ctypes.Hunk{
			{OldStart: 1, NewStart: 1},
		},
	}

	file2 := &ctypes.FileDiff{
		NewPath: "test.go",
		OldPath: "test.go",
		IsNew:   false,
		Hunks: []*ctypes.Hunk{
			{OldStart: 1, NewStart: 1},
		},
	}

	file3 := &ctypes.FileDiff{
		NewPath: "other.go",
		OldPath: "other.go",
		IsNew:   false,
		Hunks: []*ctypes.Hunk{
			{OldStart: 1, NewStart: 1},
		},
	}

	assert.True(t, filesEqual(file1, file2), "filesEqual() returned false for identical files")
	assert.False(t, filesEqual(file1, file3), "filesEqual() returned true for different files")
}
