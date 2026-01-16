package teapot

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
// Call Tick() on each compositor tick to advance animation frames.
type Ticker interface {
	Tick()
}

// RenderLogger is called to log render timing information.
// Set this to enable render logging.
var RenderLogger func(layer string, durationMs float64)

// BaseAnimationWidget provides a base implementation for animation widgets.
// Embed this in widgets that need animation support.
type BaseAnimationWidget struct {
	BaseWidget
	animBounds     []Rect
	needsAnimation bool
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

// MightBeDirty returns true for animated widgets (they always need repainting).
func (b *BaseAnimationWidget) MightBeDirty() bool {
	return true
}

// RenderInOverlay is a no-op in the base implementation.
// Override this in concrete implementations.
func (b *BaseAnimationWidget) RenderInOverlay(buf *Buffer) {
	// No-op: override in concrete implementations
}
