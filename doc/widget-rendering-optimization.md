# Widget Rendering Optimization with Observables

## Overview

This document describes an optimization approach for widget rendering that uses an observable pattern to minimize unnecessary re-renders. The key insight is that widgets should only re-render when their specific data dependencies change, rather than on every state mutation.

## Architecture

### Widget Dirty State

Widgets support a `needsRender` state mechanism:

- `needsRender` - indicates the widget needs to be re-rendered
- Cleared via `MarkRendered()` after rendering
- Set when widget state changes via `Invalidate()`

After rendering, results can be cached and reused for subsequent calls when the widget is not dirty.

### Key Methods

```go
// In Widget interface
Invalidate()        // Marks widget as needing repaint (propagates to parents)
IsDirty() bool      // Returns true if widget needs repainting
MightBeDirty() bool // Returns true if widget might need repainting

// In BaseWidget
NeedsRender() bool  // Alias for IsDirty() with clearer semantics
MarkRendered()      // Clears needsRender flag after rendering
```

## Observable-Based Rendering

### Observable Object

The app maintains an observable object that acts as a centralized state store. Widgets subscribe to specific keys in the observable rather than being directly mutated.

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
┌──────────┐     ┌────────────┐     ┌────────────┐     ┌────────┐
│  State   │────►│ Observable │────►│ Compositor │────►│ Widget │
│  Change  │     │  (notify)  │     │ (callback) │     │(render)│
└──────────┘     └────────────┘     └────────────┘     └────────┘
```

1. Application updates a key in the observable (e.g., `observable.SetValueAtKey("files", newFileList)`)
2. Observable detects the change and notifies subscribers
3. Callback is invoked with only the subscribed path (not old/new values)
4. Compositor maps the path to the appropriate widget
5. Widget is marked as `needsRender` via `Invalidate()`
6. On next render cycle, only dirty widgets re-render

## Implementation Details

### Callback Decoupling

**Critical:** The callback is NOT bound directly to the widget object.

**Problem:** Direct binding creates dependency loops and memory leaks:
```
Widget ──references──► Callback ──references──► Widget
```

**Solution:** The compositor acts as an intermediary:

```go
// Compositor subscribes on behalf of the widget
compositor.SubscribeWidget(fileListWidget, "files")
compositor.SubscribeWidget(diffViewWidget, "diff")

// Internally, compositor maintains the mapping
type widgetSubscription struct {
    widget Widget
    subID  observable.Subscription
}

type Compositor struct {
    observable          *observable.Observable
    widgetSubscriptions map[string][]widgetSubscription // path -> widgets
}

// Observable callback only receives the path
func (c *Compositor) onPathChange(path string) {
    if subs, ok := c.widgetSubscriptions[path]; ok {
        for _, sub := range subs {
            sub.widget.Invalidate()
        }
    }
}
```

This approach:
- Avoids circular references between widgets and callbacks
- Allows the compositor to manage widget lifecycle
- Enables proper cleanup when widgets are removed

### Observable Interface

The observable interface supports path-only callbacks:

```go
// PathChangeCallback receives only the changed path, not old/new values.
type PathChangeCallback func(path string)

// OnChange subscribes to changes at matching paths.
// Patterns use fnmatch-style matching.
func (o *Observable) OnChange(patterns []string, callback PathChangeCallback) Subscription

// Unsubscribe removes a subscription.
func (o *Observable) Unsubscribe(sub Subscription)
```

**Rationale for path-only callbacks:**
1. Widgets typically need to fetch fresh data anyway on re-render
2. Reduces memory overhead of storing old/new value pairs
3. Simplifies the callback signature
4. Avoids issues with value comparison for complex types

### Compositor API

```go
// SetObservable sets the observable for widget subscriptions.
func (c *Compositor) SetObservable(obs *observable.Observable)

// SubscribeWidget subscribes a widget to observable paths.
// When any path changes, the widget is invalidated.
func (c *Compositor) SubscribeWidget(widget Widget, paths ...string)

// UnsubscribeWidget removes all subscriptions for a widget.
func (c *Compositor) UnsubscribeWidget(widget Widget)
```

## Usage Example

```go
// In app initialization
obs := observable.New()
compositor := teapot.NewCompositor(mainLayout)
compositor.SetObservable(obs)

// Subscribe widgets to their data dependencies
compositor.SubscribeWidget(fileListWidget, "files")
compositor.SubscribeWidget(diffViewWidget, "diff")

// When state changes, update the observable
obs.SetValueAtKey("files", fileDiffs)  // FileListWidget auto-invalidated
obs.SetValueAtKey("diff", currentDiff) // DiffViewWidget auto-invalidated
```

## Benefits

1. **Reduced CPU usage:** Only dirty widgets re-render
2. **Predictable performance:** Render cost proportional to changes, not tree size
3. **Clear data flow:** Unidirectional data flow through observable
4. **Testability:** Observable state changes are easy to test
5. **Debugging:** Can log path changes to trace render triggers
6. **No memory leaks:** Compositor manages subscriptions, avoiding circular refs

## Implementation Status

- [x] Rename `Repaint()` to `Invalidate()` across codebase
- [x] Add `PathChangeCallback` type to Observable
- [x] Add `OnChange()` method to Observable
- [x] Add `needsRender` field and `MarkRendered()` to BaseWidget
- [x] Add widget subscription management to Compositor
- [x] Wire up FileListWidget subscription in app
- [x] Wire up DiffViewWidget subscription in app
- [ ] Migrate state updates to use observable (optional - can be done incrementally)
