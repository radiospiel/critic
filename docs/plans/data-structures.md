# Data Structures and Interfaces

> **Note:** This documentation was generated as part of a Claude Code run on 2025-12-28.

## Overview

This document describes the core data structures and interfaces used in Critic to manage diff state and UI state. These abstractions provide clear separation of concerns and enable reactive updates throughout the application.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Application Layer                        │
│                       (internal/app/app.go)                      │
└───────────────┬─────────────────────────────────┬───────────────┘
                │                                 │
                │ uses                            │ uses
                ▼                                 ▼
┌───────────────────────────────┐   ┌──────────────────────────────┐
│         DiffState             │   │        ViewState             │
│   (internal/critic/*)         │   │     (internal/ui/*)          │
├───────────────────────────────┤   ├──────────────────────────────┤
│ Interface:                    │   │ Interface:                   │
│ - GetFiles()                  │   │ - GetSelectedFile()          │
│ - GetDiffDetails(path)        │   │ - GetActiveHunkPosition()    │
│ - OnChange(callback)          │   │ - SetSelectedFile(path)      │
│ - Refresh()                   │   │ - SetActiveLine(num)         │
├───────────────────────────────┤   ├──────────────────────────────┤
│ Implementation:               │   │ Implementation:              │
│ - GitDiffState                │   │ - DefaultViewState           │
│   (wraps git.GetDiff)         │   │   (tracks UI selections)     │
└───────────────┬───────────────┘   └──────────────────────────────┘
                │
                │ uses
                ▼
┌───────────────────────────────┐
│       Git Operations          │
│    (internal/git/diff.go)     │
├───────────────────────────────┤
│ - GetDiff(paths, mode)        │
│ - GetFileContent(path, rev)   │
│ - ParseDiff(output)           │
└───────────────────────────────┘
```

## DiffState Interface

The `DiffState` interface provides access to the current state of file diffs. It abstracts away the source of diff information (currently git, but could be other sources in the future).

### Interface Definition

Located in `internal/critic/diffstate.go`:

```go
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
```

### Supporting Types

**FileState** - Represents the state of a file in the diff:
- `FileCreated` - File was newly created
- `FileDeleted` - File was deleted
- `FileChanged` - File was modified

**FileInfo** - Basic information about a changed file:
```go
type FileInfo struct {
    Path  string
    State FileState
}
```

**DiffDetails** - Detailed diff information for a file:
```go
type DiffDetails struct {
    Path            string
    Hunks           []*ctypes.HunkHeader
    OriginalContent string // Full content of original file
    CurrentContent  string // Full content of current file
}
```

**OnChangeCallback** - Called when diff details change:
```go
type OnChangeCallback func(oldDetails, newDetails *DiffDetails)
```

### GitDiffState Implementation

The `GitDiffState` struct implements the `DiffState` interface using git as the source:

**Key Features:**
- Wraps `git.GetDiff()` functionality
- Maintains callback registry for reactive updates
- Thread-safe with mutex protection
- Notifies callbacks when files are added, changed, or reverted

**Construction:**
```go
state, err := NewGitDiffState(paths []string, mode git.DiffMode)
```

**Reactive Updates:**
The `OnChange` callback system allows UI components to react to diff changes:
- Called when files are added or removed
- Called when file content changes
- Called when files are reverted to original (newDetails will be nil)

## ViewState Interface

The `ViewState` interface tracks the current UI state, including which file is selected and where the cursor is positioned within the diff view.

### Interface Definition

Located in `internal/ui/viewstate.go`:

```go
type ViewState interface {
    // GetSelectedFile returns the name/path of the currently selected file
    // Returns empty string if no file is selected
    GetSelectedFile() string

    // GetActiveHunkPosition returns the header and position that the active line is in
    // Returns nil if no file is selected or no active line
    GetActiveHunkPosition() *HunkPosition

    // SetSelectedFile sets the currently selected file
    SetSelectedFile(path string)

    // SetActiveLine sets the active line number (used to determine which header)
    SetActiveLine(lineNum int)
}
```

### Supporting Types

**HunkPosition** - Position within a header:
```go
type HunkPosition struct {
    HunkIndex        int          // Index of the header in the file
    HunkHeader             *ctypes.HunkHeader // The header itself
    LineInHunk       int          // Line number within the header (0-based)
    TotalLinesInHunk int          // Total number of lines in the header
}
```

### DefaultViewState Implementation

The `DefaultViewState` struct provides a concrete implementation:

**Key Features:**
- Tracks currently selected file path
- Tracks active line number
- Calculates which header contains the active line
- Resets active line when file selection changes

**Construction:**
```go
viewState := NewViewState()
```

## Usage Examples

### Using DiffState

```go
// Create a git-based diff state
state, err := NewGitDiffState([]string{"."}, git.DiffUnstaged)
if err != nil {
    // handle error
}

// Get list of changed files
files := state.GetFiles()
for _, file := range files {
    fmt.Printf("%s: %s\n", file.Path, file.State)
}

// Get details for a specific file
details, err := state.GetDiffDetails("main.go")
if err != nil {
    // handle error
}

// Register a callback for changes
unregister := state.OnChange(func(old, new *DiffDetails) {
    if new == nil {
        fmt.Printf("File %s was reverted\n", old.Path)
    } else {
        fmt.Printf("File %s changed\n", new.Path)
    }
})
defer unregister()

// Refresh to get latest changes
err = state.Refresh()
```

### Using ViewState

```go
// Create a view state
viewState := NewViewState()

// Set the selected file
viewState.SetSelectedFile("main.go")

// Set the active line
viewState.SetActiveLine(42)

// Get the active header position
pos := viewState.GetActiveHunkPosition()
if pos != nil {
    fmt.Printf("Line %d is in header %d (line %d of %d)\n",
        42, pos.HunkIndex, pos.LineInHunk, pos.TotalLinesInHunk)
}
```

## Benefits

### Separation of Concerns
- **DiffState** handles data acquisition and change detection
- **ViewState** handles UI selection tracking
- Clear boundaries make the code easier to understand

### Testability
- Interfaces can be mocked for unit testing
- No need to set up real git repositories in tests
- Each component can be tested in isolation

### Reactivity
- OnChange callbacks enable reactive UI updates
- UI components can subscribe to relevant changes
- Reduces coupling between components

### Extensibility
- New diff sources can implement DiffState (e.g., API-based diffs)
- ViewState can be extended with additional UI state
- Interface-based design supports future enhancements

## Implementation Status

- ✅ DiffState interface defined
- ✅ GitDiffState implementation complete
- ✅ ViewState interface defined
- ✅ DefaultViewState implementation complete
- ✅ Comprehensive test coverage
- ⏳ Integration with main application (future work)

## Related Files

- `internal/critic/diffstate.go` - DiffState interface and types
- `internal/critic/gitdiffstate.go` - GitDiffState implementation
- `internal/critic/diffstate_test.go` - Tests for DiffState
- `internal/ui/viewstate.go` - ViewState interface and implementation
- `internal/ui/viewstate_test.go` - Tests for ViewState
- `internal/git/diff.go` - Underlying git diff operations
- `pkg/types/diff.go` - Core diff data types (Diff, FileDiff, HunkHeader, Line)
