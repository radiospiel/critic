# File Watcher Architecture

> **Note:** This document is AI-generated and has not yet been human-reviewed. Please verify implementation details against the actual codebase.

## Overview

The file watcher uses a three-stage pipeline architecture to efficiently monitor file changes in the git repository. Events flow through filtering and debouncing stages before being delivered to the application.

## Pipeline Architecture

```
fsnotify.Watcher
    ↓ (file system events)
eventLoop goroutine
    ↓ (rawEvents channel)
filterLoop goroutine
    ↓ (filteredEvents channel)
debounceLoop goroutine
    ↓ (changesChan channel)
Application
```

## Components

### Watcher Struct

```go
type Watcher struct {
    watcher     *fsnotify.Watcher
    debounceMs  int
    changesChan chan FileChange

    // Filtering
    paths   []string // Git-relative paths to filter
    gitRoot string   // Git repository root

    // Pipeline channels
    rawEvents      chan fsnotify.Event  // Buffer: 100
    filteredEvents chan string          // Buffer: 100

    // Per-file debouncing
    debouncers map[string]*time.Timer  // One timer per file
    debounceMu sync.Mutex

    // Lifecycle
    stopChan chan struct{}
}
```

## Pipeline Stages

### Stage 1: eventLoop

**Purpose:** Forward file system events from fsnotify to the pipeline

**Responsibilities:**
- Receives events from `fsnotify.Watcher.Events`
- Filters for relevant operations: Write, Create, Rename, Remove
- Forwards events to `rawEvents` channel (non-blocking)
- Handles fsnotify errors
- Respects `stopChan` for graceful shutdown

**Why needed:** fsnotify emits many event types; we only care about file modifications.

### Stage 2: filterLoop

**Purpose:** Filter events based on configured paths

**Responsibilities:**
- Receives events from `rawEvents` channel
- Converts absolute paths to git-relative paths
- Checks if file matches any configured path in `w.paths`
- Forwards matching files to `filteredEvents` channel
- Logs filtered-out files for debugging

**Why needed:** When user runs `critic src/`, we watch the entire `src/` tree recursively but only care about files in the diff. Filtering reduces noise.

**Filtering Logic:**
- If no paths or `["."]` → include everything
- Otherwise, check for:
  - Exact match: `gitPath == configuredPath`
  - Directory match: `configuredPath + "/" is suffix of gitPath`
  - Trailing slash match: `configuredPath ends with "/" and is suffix of gitPath`

**Examples:**
- `critic src/` → includes `src/foo.go`, `src/bar/baz.go`
- `critic src/foo.go` → includes only `src/foo.go`
- `critic src/ tests/` → includes files in `src/` OR `tests/`
- `critic` → includes everything in current directory

### Stage 3: debounceLoop

**Purpose:** Debounce file change events per-file

**Responsibilities:**
- Receives file paths from `filteredEvents` channel
- Maintains one timer per file in `debouncers` map
- When a file changes:
  - Cancels existing timer for that file (if any)
  - Creates new timer for debounce period (default 100ms)
  - Timer fires → emits `FileChange` to `changesChan`
  - Cleans up timer from map
- Respects `stopChan` for shutdown

**Why needed:** Editors often write files multiple times in quick succession. Per-file debouncing ensures:
- Each file gets one event after changes settle
- Multiple files can be debouncing simultaneously
- No event is lost due to global debouncing

**Example:**
```
t=0ms:   file1.go changes → start 100ms timer for file1
t=20ms:  file1.go changes → cancel timer, start new 100ms timer
t=50ms:  file2.go changes → start 100ms timer for file2
t=120ms: file1 timer fires → emit FileChange{file1.go}
t=150ms: file2 timer fires → emit FileChange{file2.go}
```

Both events are delivered, each debounced independently.

## Initialization

### NewWatcher

```go
func NewWatcher(debounceMs int) (*Watcher, error)
```

**Creates:**
- fsnotify watcher
- Pipeline channels (buffered)
- Debouncer map
- Stop channel

**Starts:**
- `eventLoop` goroutine
- `filterLoop` goroutine
- `debounceLoop` goroutine

**Returns:** Ready-to-use watcher (no paths configured yet)

### WatchPaths

```go
func (w *Watcher) WatchPaths(paths []string) error
```

**Responsibilities:**
1. Store paths for filtering (convert to git-relative)
2. Watch directories recursively via fsnotify:
   - If `paths` is empty or `["."]` → watch current directory
   - If `paths` contains files → watch their parent directories
   - If `paths` contains directories → watch those directories
3. All watching is recursive (subdirectories included)

**Why watch recursively:**
- Need to detect all file changes to keep file list accurate
- Even if user specifies `critic src/foo.go`, other files in `src/` might change
- Filtering happens at the pipeline level, not at watch registration

## Lifecycle Management

### Shutdown

```go
func (w *Watcher) Close() error
```

**Steps:**
1. Close `stopChan` → signals all goroutines to exit
2. Stop all active debounce timers
3. Close fsnotify watcher

**Result:** All goroutines exit cleanly, no resource leaks

## Channel Buffer Sizes

| Channel | Size | Reason |
|---------|------|--------|
| `rawEvents` | 100 | High throughput from fsnotify |
| `filteredEvents` | 100 | Post-filter, before debounce |
| `changesChan` | 10 | Final output, consumer should be fast |

Buffers prevent event drops during burst activity.

## Benefits

### 1. Per-File Debouncing
- **Before:** Single global timer, last event wins
- **After:** One timer per file, all events delivered
- **Benefit:** No lost events when multiple files change

### 2. Path Filtering
- **Before:** Watch recursively, emit all events, app filters later
- **After:** Filter at watcher level, only emit relevant events
- **Benefit:** Reduced event noise, cleaner separation

### 3. Modularity
- **Before:** Single monolithic eventLoop
- **After:** Three focused stages with clear responsibilities
- **Benefit:** Easier to test, debug, and modify

### 4. Observability
- **Before:** Limited logging
- **After:** Detailed logging at each stage
- **Benefit:** Easy to diagnose filtering or debouncing issues

### 5. Clean Shutdown
- **Before:** Manual timer cleanup
- **After:** Coordinated shutdown via `stopChan`
- **Benefit:** No goroutine leaks, proper resource cleanup

## Usage in Critic

### When Created
Only when diffing against "current" (working directory):
```go
if args.Current == "current" {
    watcher, _ := git.NewWatcher(100) // 100ms debounce
    watcher.WatchPaths(args.Paths)
}
```

### When Not Created
Historical diffs (e.g., `HEAD~1` vs `HEAD`):
- Files in git history cannot change
- No watcher needed
- Saves resources

### Failure Handling
If watcher creation fails in "current" mode, the application **panics immediately**:
```go
if err != nil {
    panic(fmt.Sprintf("Failed to create file watcher: %v", err))
}
```

**Rationale:**
- File watching is essential in "current" mode
- Running without a watcher would result in broken functionality (UI wouldn't update)
- Fail-fast is better than degraded functionality
- Clear error message helps users diagnose the issue

**Common failure causes:**
- `fsnotify` initialization error (OS-level issue)
- Git root detection failure
- Insufficient file descriptor limits

### Event Handling
Application receives `FileChange` events:
1. Checks if changed file is currently viewed → re-render diff view
2. Reloads full diff to update file list
3. Continues waiting for next event

## Performance Considerations

### Efficient Watching
- Watch only relevant subtrees (based on `paths` argument)
- Skip `.git`, hidden directories, `node_modules`
- Recursive watching handled by fsnotify (efficient)

### Efficient Filtering
- Git-relative path conversion cached (done once per repo)
- Simple string suffix checks (O(1) per path)
- Early exit on match

### Efficient Debouncing
- Map lookup O(1)
- Timer operations O(1)
- No global lock contention (mutex only for debouncers map)

### Memory Usage
- Channels: ~100 events × ~100 bytes = ~10KB
- Debouncers: ~N active files × timer overhead = minimal
- Total: <100KB for typical usage

## Testing

### Unit Tests
Test each stage independently:
- eventLoop: Mock fsnotify events
- filterLoop: Test path matching logic
- debounceLoop: Verify timing behavior

### Integration Tests
Test full pipeline:
- Create watcher
- Modify files
- Verify correct events emitted
- Verify debouncing works
- Verify filtering works

### Manual Testing
Run with debug logging:
```bash
CRITIC_LOG_LEVEL=DEBUG critic src/
```

Watch logs for:
- Files being watched
- Events being filtered
- Debounce timers firing
