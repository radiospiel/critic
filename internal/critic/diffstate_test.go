package critic

import (
	"testing"

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
			if got := tt.state.String(); got != tt.want {
				t.Errorf("FileState.String() = %v, want %v", got, tt.want)
			}
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
					OldPath: "deleted.go",
					NewPath: "deleted.go",
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
	if len(files) != 3 {
		t.Errorf("GetFiles() returned %d files, want 3", len(files))
	}

	// Check first file (created)
	if files[0].Path != "test.go" {
		t.Errorf("First file path = %v, want test.go", files[0].Path)
	}
	if files[0].State != FileCreated {
		t.Errorf("First file state = %v, want FileCreated", files[0].State)
	}

	// Check second file (deleted)
	if files[1].Path != "deleted.go" {
		t.Errorf("Second file path = %v, want deleted.go", files[1].Path)
	}
	if files[1].State != FileDeleted {
		t.Errorf("Second file state = %v, want FileDeleted", files[1].State)
	}

	// Check third file (changed)
	if files[2].Path != "modified.go" {
		t.Errorf("Third file path = %v, want modified.go", files[2].Path)
	}
	if files[2].State != FileChanged {
		t.Errorf("Third file state = %v, want FileChanged", files[2].State)
	}
}

func TestGitDiffState_OnChange(t *testing.T) {
	state := &GitDiffState{
		diff:         &ctypes.Diff{},
		callbacks:    make(map[int]OnChangeCallback),
		nextCallbackID: 0,
	}

	called := false
	callback := func(old, new *DiffDetails) {
		called = true
	}

	// Register callback
	unregister := state.OnChange(callback)

	if len(state.callbacks) != 1 {
		t.Errorf("Expected 1 callback, got %d", len(state.callbacks))
	}

	// Unregister callback
	unregister()

	if len(state.callbacks) != 0 {
		t.Errorf("Expected 0 callbacks after unregister, got %d", len(state.callbacks))
	}

	// Callback should not have been called yet
	if called {
		t.Error("Callback was called unexpectedly")
	}
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

	if !filesEqual(file1, file2) {
		t.Error("filesEqual() returned false for identical files")
	}

	if filesEqual(file1, file3) {
		t.Error("filesEqual() returned true for different files")
	}
}
