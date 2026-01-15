# Animation Overlay Architecture

This document describes the architecture for rendering animations in the critic TUI
using an overlay-based approach.

## Overview

The animation system separates animated content from static content using a layered
rendering approach. This allows animations to be rendered efficiently on top of
static content without re-rendering the entire widget tree.

## Key Components

### 1. AnimationWidget Interface

```go
type AnimationWidget interface {
    Widget

    // RenderInOverlay renders the animated content to the overlay buffer.
    // This is called after all static content has been rendered.
    RenderInOverlay(buf *Buffer)

    // AnimationBounds returns the screen-space bounds where this widget
    // renders its animation. Used for dirty-region tracking.
    AnimationBounds() Rect

    // NeedsAnimation returns true if this widget currently has active animations.
    NeedsAnimation() bool
}
```

AnimationWidgets behave like regular widgets, but their `Render()` method renders
placeholder content (typically spaces) for the animated regions. The actual animated
content is rendered via `RenderInOverlay()`.

### 2. Overlay Widget

The Overlay widget acts as a container that manages animation rendering:

```go
type Overlay struct {
    ContainerWidget
    content          Widget              // The main content widget
    animationWidgets []AnimationWidget   // Registered animation widgets
    buffer           *Buffer             // Cached buffer from content render
    tickInterval     time.Duration       // Base tick rate (40ms)
}
```

**Responsibilities:**
- Contains the main content widget tree
- Maintains a registry of AnimationWidgets
- Renders content to a cached buffer
- Overlays animation content on each tick
- Manages tick generation and distribution

### 3. Rendering Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│                     Full Render                              │
│  1. Clear buffer                                             │
│  2. Render content widget tree (static content)              │
│  3. Cache buffer state                                       │
│  4. For each registered AnimationWidget:                     │
│     - Call RenderInOverlay() to draw animations              │
│  5. Return final buffer                                      │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                     Tick Update                              │
│  1. Reuse cached buffer (skip content re-render)             │
│  2. For each registered AnimationWidget:                     │
│     - Call RenderInOverlay() with updated frame              │
│  3. Return updated buffer                                    │
└─────────────────────────────────────────────────────────────┘
```

### 4. AnimationTicker Integration

The existing `AnimationTicker` continues to manage animation state:
- Frame advancement for different animation types
- Speed control per animation
- Color/style information

The Overlay widget owns the tick command generation instead of the app:

```go
func (o *Overlay) StartTicking() tea.Cmd {
    return tea.Tick(o.tickInterval, func(t time.Time) tea.Msg {
        return AnimationTickMsg{}
    })
}

func (o *Overlay) HandleTick() tea.Cmd {
    o.animationTicker.Tick()  // Advance frames
    o.dirty = true            // Mark for re-render
    return o.StartTicking()   // Continue ticking
}
```

## Widget Registration

AnimationWidgets register themselves with the nearest Overlay ancestor:

```go
func (w *HunkWidget) SetParent(parent Widget) {
    w.BaseWidget.SetParent(parent)

    // Find and register with overlay
    if overlay := FindOverlay(parent); overlay != nil {
        overlay.RegisterAnimation(w)
    }
}
```

Alternatively, registration can happen during widget tree construction.

## Benefits

1. **Efficient Updates**: Only animation regions are updated on tick, not the
   entire content tree.

2. **Clean Separation**: Static rendering logic is separated from animation logic.

3. **Centralized Tick Management**: The Overlay manages all animation timing,
   simplifying the app's message handling.

4. **Extensible**: New animated widgets just implement the AnimationWidget
   interface.

5. **Debuggable**: Logging in the Overlay provides visibility into animation
   rendering.

## Implementation Notes

### Placeholder Rendering

In `Render()`, AnimationWidgets render spaces or static content where animations
will appear:

```go
func (w *HunkWidget) Render(buf *SubBuffer) {
    // ... render static content ...

    // For animation regions, render placeholder spaces
    for x := 0; x < 12; x++ {
        buf.SetCell(x, separatorY, Cell{Rune: ' ', Style: defaultStyle})
    }
}
```

### Overlay Rendering

In `RenderInOverlay()`, the actual animation is drawn:

```go
func (w *HunkWidget) RenderInOverlay(buf *Buffer) {
    if w.animationTicker == nil {
        return
    }

    bounds := w.AnimationBounds()
    frame := w.animationTicker.GetSeparatorFrame()
    cells := ParseANSILine(frame)

    for x, cell := range cells {
        buf.SetCell(bounds.X+x, bounds.Y, cell)
    }
}
```

### Buffer Caching

The Overlay caches the content buffer to avoid re-rendering on tick:

```go
func (o *Overlay) Render() string {
    if o.dirty {
        o.buffer.Clear()
        o.renderContent()
        o.dirty = false
    }

    // Always render animations (they change on each tick)
    for _, aw := range o.animationWidgets {
        if aw.NeedsAnimation() {
            aw.RenderInOverlay(o.buffer)
        }
    }

    return o.buffer.String()
}
```

## File Structure

```
teapot/
  overlay.go        # Overlay widget and AnimationWidget interface
  widget.go         # Base widget (unchanged)
  compositor.go     # Integration with Overlay

internal/ui/
  animation.go      # AnimationTicker (largely unchanged)
  diffview_widgets.go  # HunkWidget implementing AnimationWidget
  filelist_widget.go   # FileListWidget implementing AnimationWidget
```

## Logging

The Overlay logs rendering activity for debugging:

```go
func (o *Overlay) Render() string {
    log.Debug().
        Int("animation_widgets", len(o.animationWidgets)).
        Bool("content_dirty", o.dirty).
        Msg("overlay render")

    // ... rendering ...
}
```
