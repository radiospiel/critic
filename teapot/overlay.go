package teapot

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// AnimationWidget is a widget that renders animated content via an overlay.
// The normal Render() method renders placeholder content (spaces), while
// RenderInOverlay() renders the actual animated content.
type AnimationWidget interface {
	Widget

	// RenderInOverlay renders the animated content directly to the buffer.
	// The buffer is the full-screen buffer, so absolute coordinates from
	// AnimationBounds() should be used.
	RenderInOverlay(buf *Buffer)

	// AnimationBounds returns the screen-space bounds where this widget
	// renders its animation. Multiple bounds can be returned if the widget
	// has multiple animation regions.
	AnimationBounds() []Rect

	// NeedsAnimation returns true if this widget currently has active animations.
	// If false, RenderInOverlay will not be called.
	NeedsAnimation() bool
}

// Ticker is an interface for animation frame advancement.
// The AnimationLayer calls Tick() on each animation tick to advance frames.
type Ticker interface {
	Tick()
}

// AnimationTickMsg is sent when it's time to update animations.
// Deprecated: Use ComposerTickMsg instead. This is kept as an alias for backwards compatibility.
type AnimationTickMsg = ComposerTickMsg

// DefaultTickInterval is the base tick rate for animations (40ms = 25fps).
// Deprecated: Use ComposerTickInterval instead.
const DefaultTickInterval = ComposerTickInterval

// RenderLogger is called to log render timing information.
// Set this to enable render logging.
var RenderLogger func(layer string, durationMs float64)

// AnimationLayer manages animation rendering on top of content.
// It contains a single content widget and a registry of animation widgets.
// On render, it first renders the content, then overlays all animations.
// The AnimationLayer owns the animation ticker and generates tick messages.
type AnimationLayer struct {
	ContainerWidget

	// content is the main widget tree to render
	content Widget

	// animationWidgets are widgets that render animated overlay content
	animationWidgets []AnimationWidget

	// cachedBuffer holds the rendered content for reuse during animation ticks
	cachedBuffer *Buffer

	// contentDirty indicates whether content needs re-rendering
	contentDirty bool

	// tickInterval is the animation tick rate
	tickInterval time.Duration

	// animationsEnabled controls whether animations are active
	animationsEnabled bool

	// ticker advances animation frames on each tick
	ticker Ticker

	// width and height of the layer
	width, height int
}

// NewAnimationLayer creates a new animation layer.
func NewAnimationLayer() *AnimationLayer {
	a := &AnimationLayer{
		ContainerWidget:   NewContainerWidget(),
		tickInterval:      DefaultTickInterval,
		animationsEnabled: true,
		contentDirty:      true,
	}
	a.SetFocusable(false)
	a.SetZOrder(ZOrderAnimation) // Animation layers render at animation z-order
	return a
}

// MightBeDirty returns true if this layer might need repainting.
// For AnimationLayer, this always returns true when animations are enabled
// (animated widgets always need repainting to show updated frames).
func (a *AnimationLayer) MightBeDirty() bool {
	return a.animationsEnabled || a.IsDirty()
}

// SetContent sets the main content widget.
func (a *AnimationLayer) SetContent(w Widget) {
	if a.content != nil {
		a.content.SetParent(nil)
	}
	a.content = w
	if w != nil {
		w.SetParent(a)
		w.SetBounds(Rect{X: 0, Y: 0, Width: a.width, Height: a.height})
	}
	a.contentDirty = true
}

// Content returns the main content widget.
func (a *AnimationLayer) Content() Widget {
	return a.content
}

// SetTicker sets the animation ticker.
// The ticker's Tick() method is called on each animation tick.
// This also sets the global ticker so widgets can access it via GlobalTicker().
func (a *AnimationLayer) SetTicker(t Ticker) {
	a.ticker = t
	globalTicker = t
}

// Ticker returns the animation ticker.
func (a *AnimationLayer) Ticker() Ticker {
	return a.ticker
}

// RegisterAnimation adds an animation widget to the layer.
func (a *AnimationLayer) RegisterAnimation(aw AnimationWidget) {
	// Check if already registered
	for _, existing := range a.animationWidgets {
		if existing == aw {
			return
		}
	}
	a.animationWidgets = append(a.animationWidgets, aw)
}

// UnregisterAnimation removes an animation widget from the layer.
func (a *AnimationLayer) UnregisterAnimation(aw AnimationWidget) {
	for i, existing := range a.animationWidgets {
		if existing == aw {
			a.animationWidgets = append(a.animationWidgets[:i], a.animationWidgets[i+1:]...)
			return
		}
	}
}

// ClearAnimations removes all registered animation widgets.
func (a *AnimationLayer) ClearAnimations() {
	a.animationWidgets = nil
}

// AnimationWidgets returns the registered animation widgets.
func (a *AnimationLayer) AnimationWidgets() []AnimationWidget {
	return a.animationWidgets
}

// SetAnimationsEnabled enables or disables animation rendering.
func (a *AnimationLayer) SetAnimationsEnabled(enabled bool) {
	a.animationsEnabled = enabled
}

// AnimationsEnabled returns whether animations are enabled.
func (a *AnimationLayer) AnimationsEnabled() bool {
	return a.animationsEnabled
}

// SetTickInterval sets the animation tick interval.
func (a *AnimationLayer) SetTickInterval(d time.Duration) {
	a.tickInterval = d
}

// TickInterval returns the animation tick interval.
func (a *AnimationLayer) TickInterval() time.Duration {
	return a.tickInterval
}

// MarkContentDirty marks the content as needing re-rendering.
func (a *AnimationLayer) MarkContentDirty() {
	a.contentDirty = true
}

// SetBounds sets the layer bounds and propagates to content.
func (a *AnimationLayer) SetBounds(bounds Rect) {
	a.bounds = bounds
	a.width = bounds.Width
	a.height = bounds.Height

	// Reallocate buffer if size changed
	if a.cachedBuffer == nil || a.cachedBuffer.Width() != bounds.Width || a.cachedBuffer.Height() != bounds.Height {
		a.cachedBuffer = NewBuffer(bounds.Width, bounds.Height)
		a.contentDirty = true
	}

	// Propagate to content
	if a.content != nil {
		a.content.SetBounds(Rect{X: 0, Y: 0, Width: bounds.Width, Height: bounds.Height})
	}
}

// Children returns the layer's children (just the content widget).
func (a *AnimationLayer) Children() []Widget {
	if a.content != nil {
		return []Widget{a.content}
	}
	return nil
}

// Render renders the layer: content first, then animation overlays.
func (a *AnimationLayer) Render(buf *SubBuffer) {
	// Render content if dirty
	if a.contentDirty && a.content != nil {
		start := time.Now()
		a.cachedBuffer.Clear()
		contentSub := a.cachedBuffer.Sub(a.cachedBuffer.Bounds())
		RenderWidget(a.content, contentSub)
		a.contentDirty = false
		if RenderLogger != nil {
			RenderLogger("baselayer", float64(time.Since(start).Microseconds())/1000.0)
		}
	}

	// Copy cached buffer to output
	if a.cachedBuffer != nil {
		buf.parent.Blit(a.cachedBuffer, buf.offset.X, buf.offset.Y)
	}

	// Render animation overlays
	if a.animationsEnabled && a.cachedBuffer != nil {
		start := time.Now()
		rendered := false
		for _, aw := range a.animationWidgets {
			if aw.NeedsAnimation() {
				aw.RenderInOverlay(buf.parent)
				rendered = true
			}
		}
		if rendered && RenderLogger != nil {
			RenderLogger("animation", float64(time.Since(start).Microseconds())/1000.0)
		}
	}
}

// RenderToBuffer renders directly to a buffer and returns it.
// This is useful for the compositor integration.
func (a *AnimationLayer) RenderToBuffer() *Buffer {
	if a.cachedBuffer == nil {
		a.cachedBuffer = NewBuffer(a.width, a.height)
	}

	// Render content if dirty
	if a.contentDirty && a.content != nil {
		start := time.Now()
		a.cachedBuffer.Clear()
		contentSub := a.cachedBuffer.Sub(a.cachedBuffer.Bounds())
		RenderWidget(a.content, contentSub)
		a.contentDirty = false
		if RenderLogger != nil {
			RenderLogger("baselayer", float64(time.Since(start).Microseconds())/1000.0)
		}
	}

	// Create output buffer (copy of cached)
	output := a.cachedBuffer.Clone()

	// Render animation overlays
	if a.animationsEnabled {
		start := time.Now()
		rendered := false
		for _, aw := range a.animationWidgets {
			if aw.NeedsAnimation() {
				aw.RenderInOverlay(output)
				rendered = true
			}
		}
		if rendered && RenderLogger != nil {
			RenderLogger("animation", float64(time.Since(start).Microseconds())/1000.0)
		}
	}

	return output
}

// StartTicking returns a command that starts the animation tick loop.
// Deprecated: Use Compositor.StartTicking() instead. The compositor now manages all ticks.
func (a *AnimationLayer) StartTicking() tea.Cmd {
	if !a.animationsEnabled {
		return nil
	}
	return tea.Tick(a.tickInterval, func(t time.Time) tea.Msg {
		return ComposerTickMsg{}
	})
}

// HandleTick processes an animation tick: advances the ticker and continues.
// Returns a command to continue ticking.
// Deprecated: Use Compositor.HandleTick() instead. The compositor now manages all ticks.
func (a *AnimationLayer) HandleTick() tea.Cmd {
	if !a.animationsEnabled {
		return nil
	}
	// Advance the ticker
	if a.ticker != nil {
		a.ticker.Tick()
	}
	// Mark this layer as needing repaint for animation frames
	a.Repaint()
	// Continue ticking
	return a.StartTicking()
}

// HandleKey routes key events to the content widget.
func (a *AnimationLayer) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if a.content != nil {
		return a.content.HandleKey(msg)
	}
	return false, nil
}

// HandleMouse routes mouse events to the content widget.
func (a *AnimationLayer) HandleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	if a.content != nil {
		return a.content.HandleMouse(msg)
	}
	return false, nil
}

// BaseAnimationWidget provides a base implementation for animation widgets.
// Embed this in widgets that need animation support.
type BaseAnimationWidget struct {
	BaseWidget
	layer          *AnimationLayer
	animBounds     []Rect
	needsAnimation bool
}

// SetAnimationLayer sets the layer this widget is registered with.
func (b *BaseAnimationWidget) SetAnimationLayer(a *AnimationLayer) {
	// Unregister from old layer
	if b.layer != nil {
		b.layer.UnregisterAnimation(b)
	}
	b.layer = a
}

// AnimationLayer returns the layer this widget is registered with.
func (b *BaseAnimationWidget) AnimationLayer() *AnimationLayer {
	return b.layer
}

// SetAnimationBounds sets the animation bounds.
func (b *BaseAnimationWidget) SetAnimationBounds(bounds []Rect) {
	b.animBounds = bounds
}

// AnimationBounds returns the animation bounds.
func (b *BaseAnimationWidget) AnimationBounds() []Rect {
	return b.animBounds
}

// SetNeedsAnimation sets whether this widget needs animation.
func (b *BaseAnimationWidget) SetNeedsAnimation(needs bool) {
	b.needsAnimation = needs
}

// NeedsAnimation returns whether this widget needs animation.
func (b *BaseAnimationWidget) NeedsAnimation() bool {
	return b.needsAnimation
}

// RenderInOverlay is a no-op in the base implementation.
// Override this in concrete implementations.
func (b *BaseAnimationWidget) RenderInOverlay(buf *Buffer) {
	// No-op: override in concrete implementations
}

// FindAnimationLayer walks up the widget tree to find the nearest AnimationLayer.
func FindAnimationLayer(w Widget) *AnimationLayer {
	for w != nil {
		if a, ok := w.(*AnimationLayer); ok {
			return a
		}
		w = w.Parent()
	}
	return nil
}
