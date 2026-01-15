# Animation Layer Architecture

This document describes the architecture for rendering animations in the critic TUI
using a layered rendering approach.

## Overview

The animation system separates animated content from static content using a layered
approach. This allows animations to be rendered efficiently on top of static content
without re-rendering the entire widget tree.

## Key Components

### 1. AnimationWidget Interface

```go
type AnimationWidget interface {
    Widget

    // RenderInOverlay renders the animated content to the buffer.
    // This is called after all static content has been rendered.
    RenderInOverlay(buf *Buffer)

    // AnimationBounds returns the screen-space bounds where this widget
    // renders its animation. Used for dirty-region tracking.
    AnimationBounds() []Rect

    // NeedsAnimation returns true if this widget currently has active animations.
    NeedsAnimation() bool
}
```

AnimationWidgets behave like regular widgets, but their `Render()` method renders
placeholder content (typically spaces) for the animated regions. The actual animated
content is rendered via `RenderInOverlay()`.

### 2. Ticker Interface

```go
type Ticker interface {
    Tick()
}
```

The Ticker interface allows AnimationLayer to advance animation frames without
depending on the specific AnimationTicker implementation.

### 3. AnimationLayer

The AnimationLayer manages animation rendering on top of content:

```go
type AnimationLayer struct {
    ContainerWidget
    content          Widget              // The main content widget
    animationWidgets []AnimationWidget   // Registered animation widgets
    cachedBuffer     *Buffer             // Cached buffer from content render
    ticker           Ticker              // Animation ticker (implements Tick())
    tickInterval     time.Duration       // Base tick rate (40ms)
}
```

**Responsibilities:**
- Contains the main content widget tree
- Maintains a registry of AnimationWidgets
- Owns the animation ticker
- Renders content to a cached buffer
- Overlays animation content on each tick
- Manages tick generation and distribution
- Logs render timing

### 4. Rendering Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│                     Full Render                              │
│  1. Clear buffer                                             │
│  2. Render content widget tree (static content)              │
│  3. Cache buffer state                                       │
│  4. Log: render (baselayer): X.X ms                          │
│  5. For each registered AnimationWidget:                     │
│     - Call RenderInOverlay() to draw animations              │
│  6. Log: render (animation): X.X ms                          │
│  7. Return final buffer                                      │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                     Tick Update                              │
│  1. AnimationLayer.HandleTick() is called                    │
│  2. Ticker.Tick() advances animation frames                  │
│  3. Continue tick loop via StartTicking()                    │
│  4. View re-renders with new animation frames                │
└─────────────────────────────────────────────────────────────┘
```

### 5. AnimationTicker Integration

The `AnimationTicker` in `internal/ui/animation.go` implements the `Ticker` interface:
- Frame advancement for different animation types (thinking, lookHere, separator)
- Speed control per animation
- Color/style information for rendering

The AnimationLayer owns the tick command generation:

```go
func (a *AnimationLayer) StartTicking() tea.Cmd {
    return tea.Tick(a.tickInterval, func(t time.Time) tea.Msg {
        return AnimationTickMsg{}
    })
}

func (a *AnimationLayer) HandleTick() tea.Cmd {
    if a.ticker != nil {
        a.ticker.Tick()  // Advance frames
    }
    return a.StartTicking()  // Continue ticking
}
```

## Widget Registration

AnimationWidgets can register with an AnimationLayer:

```go
animLayer.RegisterAnimation(hunkWidget)
animLayer.RegisterAnimation(fileListWidget)
```

Or find the nearest AnimationLayer in the widget tree:

```go
if layer := FindAnimationLayer(parent); layer != nil {
    layer.RegisterAnimation(w)
}
```

## Benefits

1. **Centralized Tick Management**: The AnimationLayer manages all animation timing,
   simplifying the app's message handling.

2. **Clean Separation**: Static rendering logic is separated from animation logic.

3. **Extensible**: New animated widgets just implement the AnimationWidget interface.

4. **Debuggable**: Logging provides visibility into render timing:
   - `render (baselayer): X.X ms` - time to render static content
   - `render (animation): X.X ms` - time to render animation overlays

5. **Future Optimization**: The cached buffer enables skipping base layer re-render
   when only animations change.

## Implementation Notes

### Placeholder Rendering

In `Render()`, AnimationWidgets render spaces where animations will appear:

```go
func (w *HunkWidget) Render(buf *SubBuffer) {
    // ... render static content ...

    // For animation regions, render placeholder spaces
    for x := 0; x < 12; x++ {
        buf.SetCell(x, separatorY, Cell{Rune: ' ', Style: defaultStyle})
    }
}
```

### Animation Overlay Rendering

In `RenderInOverlay()`, the actual animation is drawn:

```go
func (w *HunkWidget) RenderInOverlay(buf *Buffer) {
    if w.animationTicker == nil {
        return
    }

    for _, bounds := range w.animBounds {
        frame := w.animationTicker.GetSeparatorFrame()
        cells := ParseANSILine(frame)
        for x, cell := range cells {
            buf.SetCell(bounds.X+x, bounds.Y, cell)
        }
    }
}
```

### Position Tracking

Widgets track their absolute screen position during `Render()` using `SubBuffer.AbsoluteOffset()`:

```go
func (w *FileListWidget) renderItem(buf *pot.SubBuffer, ...) {
    absX, absY := buf.AbsoluteOffset()
    w.animInfos = append(w.animInfos, fileAnimInfo{
        bounds: pot.Rect{X: absX, Y: absY, Width: 1, Height: 1},
        state:  animState,
    })
}
```

## File Structure

```
teapot/
  overlay.go        # AnimationLayer, AnimationWidget interface, Ticker interface
  widget.go         # Base widget (unchanged)
  buffer.go         # Buffer with AbsoluteOffset() method

internal/ui/
  animation.go      # AnimationTicker (implements Ticker interface)
  diffview_widgets.go  # HunkWidget, DiffViewWidget implementing AnimationWidget
  filelist_widget.go   # FileListWidget implementing AnimationWidget

internal/app/
  app.go            # Creates AnimationLayer, handles AnimationTickMsg
```

## Logging

Set `teapot.RenderLogger` to enable render timing:

```go
teapot.RenderLogger = func(layer string, durationMs float64) {
    logger.Info("render (%s): %.1f ms", layer, durationMs)
}
```

Output:
```
render (baselayer): 12.3 ms
render (animation): 0.1 ms
```
