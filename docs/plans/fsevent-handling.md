# Plan: File System Event Handling

## Goal

Ensure that changes are reloaded when:
1. A file changes on disk that is currently shown in the diff view
2. A symbolic ref (branch, tag, HEAD) changes due to commits

## Current State

### Existing Components

**File Watcher** (`internal/git/watcher.go`):
- Three-stage pipeline: `fsnotify → eventLoop → filterLoop → debounceLoop`
- Per-file debouncing (configurable, default 100ms)
- Filters events by configured paths
- Emits `FileChange` events via channel
- Skips `.git`, hidden directories, `node_modules`

**Base Resolver** (`internal/git/baseresolver.go`):
- Polls every 10 seconds to detect ref changes
- Resolves `merge-base`, branch names, commit SHAs
- Calls `onChange` callback when any base changes
- Already integrated into app via `baseChangedMsg`

### Gaps

1. **File watcher not integrated into app** - The watcher exists but isn't used by the application model
2. **No detection of symbolic ref changes** - The watcher doesn't watch `.git/` directory for ref updates
3. **No targeted refresh** - When a file changes, the entire diff is reloaded rather than just refreshing the affected file

## Implementation Plan

### Phase 1: Integrate File Watcher into App

**Files to modify:**
- `internal/app/app.go`

**Steps:**

1. Add watcher field to Model:
   ```go
   type Model struct {
       // existing fields...
       watcher *git.Watcher
   }
   ```

2. Initialize watcher in `NewModel`:
   ```go
   watcher, err := git.NewWatcher(100) // 100ms debounce
   if err != nil {
       logger.Error("Failed to create watcher: %v", err)
       // Continue without watcher (graceful degradation)
   } else {
       watcher.WatchPaths(args.Paths)
   }
   ```

3. Create message type for file changes:
   ```go
   type fileChangedMsg struct {
       Path string
   }
   ```

4. Subscribe to watcher in `Init()`:
   ```go
   func (m Model) Init() tea.Cmd {
       return tea.Batch(
           // existing commands...
           m.watchForFileChanges(),
       )
   }

   func (m *Model) watchForFileChanges() tea.Cmd {
       if m.watcher == nil {
           return nil
       }
       return func() tea.Msg {
           change := <-m.watcher.Changes()
           return fileChangedMsg{Path: change.Path}
       }
   }
   ```

5. Handle file change in `Update()`:
   ```go
   case fileChangedMsg:
       logger.Info("File changed: %s", msg.Path)
       // Reload diff for changed file
       return m, tea.Batch(
           m.loadDiffCmd(),
           m.watchForFileChanges(), // Continue watching
       )
   ```

6. Clean up watcher on exit (in `Update` for `tea.Quit`):
   ```go
   if m.watcher != nil {
       m.watcher.Close()
   }
   ```

### Phase 2: Watch for Git Ref Changes

**Goal:** Detect when HEAD, branches, or tags change due to commits, merges, rebases, etc.

**Files to modify:**
- `internal/git/watcher.go` (add ref watching)
- `internal/app/app.go` (handle ref changes)

**Approach:** Watch `.git/` directory for changes to:
- `.git/HEAD` - Current branch pointer
- `.git/refs/heads/*` - Local branches
- `.git/refs/tags/*` - Tags
- `.git/refs/remotes/*` - Remote tracking refs
- `.git/FETCH_HEAD` - After fetch operations
- `.git/ORIG_HEAD` - After rebase/merge

**Steps:**

1. Add git ref watcher to `Watcher` struct:
   ```go
   type Watcher struct {
       // existing fields...
       gitRefEvents chan GitRefChange
   }

   type GitRefChange struct {
       RefPath string // e.g., "refs/heads/main"
   }
   ```

2. Add method to watch git refs:
   ```go
   func (w *Watcher) WatchGitRefs() error {
       gitDir := filepath.Join(w.gitRoot, ".git")

       // Watch specific git directories
       paths := []string{
           gitDir,
           filepath.Join(gitDir, "refs", "heads"),
           filepath.Join(gitDir, "refs", "tags"),
           filepath.Join(gitDir, "refs", "remotes"),
       }

       for _, path := range paths {
           if err := w.watcher.Add(path); err != nil {
               // Directory might not exist yet, that's OK
               logger.Debug("Could not watch %s: %v", path, err)
           }
       }
       return nil
   }
   ```

3. Update `eventLoop` to detect git ref changes:
   ```go
   // In eventLoop, check if changed file is in .git/
   if strings.Contains(event.Name, "/.git/") {
       if isGitRefFile(event.Name) {
           w.gitRefEvents <- GitRefChange{RefPath: event.Name}
       }
   }
   ```

4. Add helper to identify ref files:
   ```go
   func isGitRefFile(path string) bool {
       return strings.Contains(path, "/refs/") ||
              strings.HasSuffix(path, "/HEAD") ||
              strings.HasSuffix(path, "/FETCH_HEAD") ||
              strings.HasSuffix(path, "/ORIG_HEAD")
   }
   ```

5. Handle in app:
   ```go
   type gitRefChangedMsg struct{}

   func (m *Model) watchForGitRefChanges() tea.Cmd {
       return func() tea.Msg {
           <-m.watcher.GitRefChanges()
           return gitRefChangedMsg{}
       }
   }

   case gitRefChangedMsg:
       logger.Info("Git ref changed, reloading")
       return m, tea.Batch(
           m.loadDiffCmd(),
           m.watchForGitRefChanges(),
       )
   ```

### Phase 3: Optimize Refresh (Optional Enhancement)

**Goal:** Only refresh what's needed instead of reloading entire diff.

**For file changes:**
- If changed file is in the current diff, reload only that file's diff
- If changed file is not in the diff, check if it should now be included
- Update file list indicators without full reload

**For ref changes:**
- Only reload if the changed ref is one of the configured bases
- Cache resolved SHAs to detect actual changes

**Implementation:**
```go
case fileChangedMsg:
    gitPath := git.AbsPathToGitPath(msg.Path)

    // Check if this file is currently displayed
    if m.diffContainsFile(gitPath) {
        // Reload just this file's diff
        return m, m.reloadFileDiffCmd(gitPath)
    }

    // Check if file should now be in diff (new file matching filters)
    if m.shouldIncludeFile(gitPath) {
        return m, m.loadDiffCmd()
    }

    // File not relevant, continue watching
    return m, m.watchForFileChanges()
```

## Testing

### Unit Tests

1. **Watcher git ref detection:**
   ```go
   func TestWatcher_DetectsGitRefChanges(t *testing.T) {
       // Create temp git repo
       // Initialize watcher with WatchGitRefs()
       // Modify .git/refs/heads/main
       // Assert GitRefChange received
   }
   ```

2. **Filter for ref files:**
   ```go
   func TestIsGitRefFile(t *testing.T) {
       assert.True(t, isGitRefFile("/repo/.git/HEAD"))
       assert.True(t, isGitRefFile("/repo/.git/refs/heads/main"))
       assert.False(t, isGitRefFile("/repo/.git/objects/ab/cd1234"))
   }
   ```

### Integration Tests

1. **End-to-end file change:**
   - Start critic on a repo
   - Modify a file in the diff
   - Assert diff view updates

2. **End-to-end commit:**
   - Start critic on a repo
   - Make a commit
   - Assert diff view updates to reflect new HEAD

### Manual Testing

```bash
# Terminal 1: Run critic
critic main..current

# Terminal 2: Make changes
echo "test" >> some_file.go  # Should trigger reload
git commit -am "test"         # Should trigger reload
git fetch                     # Should trigger reload if remote changed
```

## Considerations

### Performance

- Debouncing prevents excessive reloads during rapid edits
- Git ref changes are less frequent, no debouncing needed
- Consider rate-limiting ref change responses (e.g., max once per second)

### Edge Cases

- Watcher creation fails (e.g., too many file descriptors) - graceful degradation
- Git ref files don't exist yet (new repo) - handle ENOENT
- Rapid ref changes (e.g., during rebase) - debounce or coalesce
- File renamed - currently handled by RENAME event

### Future Enhancements

- Watch for `.gitignore` changes to update file filtering
- Watch for stash operations (`.git/refs/stash`)
- Provide visual indicator when reload is in progress
- Allow user to manually trigger refresh (e.g., `r` key)
