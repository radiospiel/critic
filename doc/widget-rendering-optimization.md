# Widget Rendering Optimization with Observables

## Overview

This document describes an optimization approach for widget rendering that uses an observable pattern to minimize unnecessary re-renders. The key insight is that widgets should only re-render when their specific data dependencies change, rather than on every state mutation.

## Current Architecture

### Widget Dirty State

Widgets currently support a dirty state mechanism:

- `needsRender` - indicates the widget needs to be re-rendered
- Cleared after `Render()` is called
- Set when widget state changes via `Repaint()`

After rendering, results are cached and reused for subsequent calls when the widget is not dirty.

### Current Limitations

1. Every `Repaint()` call propagates up to the root widget
2. The compositor checks the entire widget tree on each tick
3. No selective invalidation based on what data actually changed
4. Widgets re-render even when their specific data hasn't changed

## Proposed Solution: Observable-Based Rendering

### Observable Object

Add an observable object to the app that acts as a centralized state store. Widgets subscribe to specific keys in the observable rather than being directly mutated.

```
┌─────────────────────────────────────────────────────────┐
│                      Observable                          │
│                                                          │
│   "files"  ──────────────►  FileList structure          │
│   "diff"   ──────────────►  Diff structure              │
│   "status" ──────────────►  Status structure            │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Widget Subscriptions

Each widget subscribes to one or more keys:

| Widget          | Observed Key(s) |
|-----------------|-----------------|
| FileListWidget  | `"files"`       |
| DiffViewWidget  | `"diff"`        |
| StatusBar       | `"status"`      |

### Change Detection Flow

```
┌──────────┐     ┌────────────┐     ┌───────────┐     ┌────────┐
│  State   │────►│ Observable │────►│ Composer  │────►│ Widget │
│  Change  │     │  (notify)  │     │ (callback)│     │(render)│
└──────────┘     └────────────┘     └───────────┘     └────────┘
```

1. Application updates a key in the observable (e.g., `observable.Set("files", newFileList)`)
2. Observable detects the change and notifies subscribers
3. Callback is invoked with only the subscribed path (not old/new values)
4. Composer maps the path to the appropriate widget
5. Widget is marked as `needsRender`
6. On next render cycle, only dirty widgets re-render

## Key Design Decisions

### Callback Decoupling

**Critical:** The callback must NOT be bound directly to the widget object.

**Problem:** Direct binding creates dependency loops and memory leaks:
```
Widget ──references──► Callback ──references──► Widget
```

**Solution:** The composer acts as an intermediary:

```go
// Composer subscribes on behalf of the widget
composer.subscribeWidget(widget, "files")

// Internally, composer maintains the mapping
type Composer struct {
    observable    *Observable
    subscriptions map[string]Widget  // path -> widget
}

// Observable callback only receives the path
func (c *Composer) onPathChange(path string) {
    if widget, ok := c.subscriptions[path]; ok {
        widget.SetNeedsRender()
    }
}
```

This approach:
- Avoids circular references between widgets and callbacks
- Allows the composer to manage widget lifecycle
- Enables proper cleanup when widgets are removed

### Observable Interface

The observable interface yields only the subscribed path, not old and new values:

```go
type PathChangeCallback func(path string)

type Observable interface {
    // Set a value at the given path
    Set(path string, value any)

    // Get a value at the given path
    Get(path string) any

    // Subscribe to changes at a path
    Subscribe(path string, callback PathChangeCallback) Subscription

    // Unsubscribe
    Unsubscribe(subscription Subscription)
}
```

**Rationale for path-only callbacks:**
1. Widgets typically need to fetch fresh data anyway on re-render
2. Reduces memory overhead of storing old/new value pairs
3. Simplifies the callback signature
4. Avoids issues with value comparison for complex types

## Data Structures

### Files Observable Value

```go
type FilesState struct {
    Files       []*FileDiff
    SelectedIdx int
}
```

### Diff Observable Value

```go
type DiffState struct {
    File          *FileDiff
    HighlightedLines []HighlightedLine
    CursorPosition   int
    YOffset          int
}
```

## Composer Responsibilities

The composer manages the widget-observable relationship:

1. **Registration:** When a widget is added, composer registers its subscriptions
2. **Mapping:** Maintains a map from observable paths to widgets
3. **Notification:** Receives path change notifications and marks appropriate widgets dirty
4. **Cleanup:** Unsubscribes when widgets are removed

```go
type Composer struct {
    root        Widget
    observable  *Observable
    pathWidgets map[string][]Widget
}

func (c *Composer) RegisterWidget(w Widget, paths ...string) {
    for _, path := range paths {
        c.observable.Subscribe(path, func(changedPath string) {
            c.markWidgetDirty(changedPath)
        })
        c.pathWidgets[path] = append(c.pathWidgets[path], w)
    }
}

func (c *Composer) markWidgetDirty(path string) {
    for _, widget := range c.pathWidgets[path] {
        widget.SetNeedsRender()
    }
}
```

## Render Caching

Widgets cache their render output:

```go
type CachedWidget struct {
    BaseWidget
    needsRender bool
    cachedBuffer *Buffer
}

func (w *CachedWidget) Render(buf *SubBuffer) {
    if !w.needsRender && w.cachedBuffer != nil {
        // Reuse cached render
        buf.CopyFrom(w.cachedBuffer)
        return
    }

    // Perform actual render
    w.doRender(buf)

    // Cache the result
    w.cachedBuffer = buf.Copy()
    w.needsRender = false
}
```

## Benefits

1. **Reduced CPU usage:** Only dirty widgets re-render
2. **Predictable performance:** Render cost proportional to changes, not tree size
3. **Clear data flow:** Unidirectional data flow through observable
4. **Testability:** Observable state changes are easy to test
5. **Debugging:** Can log path changes to trace render triggers

## Implementation Steps

1. Modify the existing `Observable` interface to use path-only callbacks
2. Add observable instance to the app
3. Update `Composer` to manage widget subscriptions
4. Add `needsRender` state and caching to base widget
5. Update `FileListWidget` to subscribe to `"files"`
6. Update `DiffViewWidget` to subscribe to `"diff"`
7. Migrate state updates to go through the observable
8. Add tests for the new rendering behavior
