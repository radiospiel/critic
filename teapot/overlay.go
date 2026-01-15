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

// AnimationTickMsg is sent when it's time to update animations.
type AnimationTickMsg struct{}

// DefaultTickInterval is the base tick rate for animations (40ms = 25fps).
const DefaultTickInterval = 40 * time.Millisecond

// Overlay is a widget that manages animation rendering on top of content.
// It contains a single content widget and a registry of animation widgets.
// On render, it first renders the content, then overlays all animations.
type Overlay struct {
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

	// width and height of the overlay
	width, height int
}

// NewOverlay creates a new overlay widget.
func NewOverlay() *Overlay {
	o := &Overlay{
		ContainerWidget:   NewContainerWidget(),
		tickInterval:      DefaultTickInterval,
		animationsEnabled: true,
		contentDirty:      true,
	}
	o.SetFocusable(false)
	return o
}

// SetContent sets the main content widget.
func (o *Overlay) SetContent(w Widget) {
	if o.content != nil {
		o.content.SetParent(nil)
	}
	o.content = w
	if w != nil {
		w.SetParent(o)
		w.SetBounds(Rect{X: 0, Y: 0, Width: o.width, Height: o.height})
	}
	o.contentDirty = true
}

// Content returns the main content widget.
func (o *Overlay) Content() Widget {
	return o.content
}

// RegisterAnimation adds an animation widget to the overlay.
func (o *Overlay) RegisterAnimation(aw AnimationWidget) {
	// Check if already registered
	for _, existing := range o.animationWidgets {
		if existing == aw {
			return
		}
	}
	o.animationWidgets = append(o.animationWidgets, aw)
}

// UnregisterAnimation removes an animation widget from the overlay.
func (o *Overlay) UnregisterAnimation(aw AnimationWidget) {
	for i, existing := range o.animationWidgets {
		if existing == aw {
			o.animationWidgets = append(o.animationWidgets[:i], o.animationWidgets[i+1:]...)
			return
		}
	}
}

// ClearAnimations removes all registered animation widgets.
func (o *Overlay) ClearAnimations() {
	o.animationWidgets = nil
}

// AnimationWidgets returns the registered animation widgets.
func (o *Overlay) AnimationWidgets() []AnimationWidget {
	return o.animationWidgets
}

// SetAnimationsEnabled enables or disables animation rendering.
func (o *Overlay) SetAnimationsEnabled(enabled bool) {
	o.animationsEnabled = enabled
}

// AnimationsEnabled returns whether animations are enabled.
func (o *Overlay) AnimationsEnabled() bool {
	return o.animationsEnabled
}

// SetTickInterval sets the animation tick interval.
func (o *Overlay) SetTickInterval(d time.Duration) {
	o.tickInterval = d
}

// TickInterval returns the animation tick interval.
func (o *Overlay) TickInterval() time.Duration {
	return o.tickInterval
}

// MarkContentDirty marks the content as needing re-rendering.
func (o *Overlay) MarkContentDirty() {
	o.contentDirty = true
}

// SetBounds sets the overlay bounds and propagates to content.
func (o *Overlay) SetBounds(bounds Rect) {
	o.bounds = bounds
	o.width = bounds.Width
	o.height = bounds.Height

	// Reallocate buffer if size changed
	if o.cachedBuffer == nil || o.cachedBuffer.Width() != bounds.Width || o.cachedBuffer.Height() != bounds.Height {
		o.cachedBuffer = NewBuffer(bounds.Width, bounds.Height)
		o.contentDirty = true
	}

	// Propagate to content
	if o.content != nil {
		o.content.SetBounds(Rect{X: 0, Y: 0, Width: bounds.Width, Height: bounds.Height})
	}
}

// Children returns the overlay's children (just the content widget).
func (o *Overlay) Children() []Widget {
	if o.content != nil {
		return []Widget{o.content}
	}
	return nil
}

// Render renders the overlay: content first, then animation overlays.
func (o *Overlay) Render(buf *SubBuffer) {
	// Render content if dirty
	if o.contentDirty && o.content != nil {
		o.cachedBuffer.Clear()
		contentSub := o.cachedBuffer.Sub(o.cachedBuffer.Bounds())
		RenderWidget(o.content, contentSub)
		o.contentDirty = false
	}

	// Copy cached buffer to output
	if o.cachedBuffer != nil {
		buf.parent.Blit(o.cachedBuffer, buf.offset.X, buf.offset.Y)
	}

	// Render animation overlays
	if o.animationsEnabled && o.cachedBuffer != nil {
		for _, aw := range o.animationWidgets {
			if aw.NeedsAnimation() {
				// Render to the parent buffer at the correct offset
				aw.RenderInOverlay(buf.parent)
			}
		}
	}
}

// RenderToBuffer renders directly to a buffer and returns it.
// This is useful for the compositor integration.
func (o *Overlay) RenderToBuffer() *Buffer {
	if o.cachedBuffer == nil {
		o.cachedBuffer = NewBuffer(o.width, o.height)
	}

	// Render content if dirty
	if o.contentDirty && o.content != nil {
		o.cachedBuffer.Clear()
		contentSub := o.cachedBuffer.Sub(o.cachedBuffer.Bounds())
		RenderWidget(o.content, contentSub)
		o.contentDirty = false
	}

	// Create output buffer (copy of cached)
	output := o.cachedBuffer.Clone()

	// Render animation overlays
	if o.animationsEnabled {
		for _, aw := range o.animationWidgets {
			if aw.NeedsAnimation() {
				aw.RenderInOverlay(output)
			}
		}
	}

	return output
}

// StartTicking returns a command that starts the animation tick loop.
func (o *Overlay) StartTicking() tea.Cmd {
	if !o.animationsEnabled {
		return nil
	}
	return tea.Tick(o.tickInterval, func(t time.Time) tea.Msg {
		return AnimationTickMsg{}
	})
}

// HandleKey routes key events to the content widget.
func (o *Overlay) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if o.content != nil {
		return o.content.HandleKey(msg)
	}
	return false, nil
}

// HandleMouse routes mouse events to the content widget.
func (o *Overlay) HandleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	if o.content != nil {
		return o.content.HandleMouse(msg)
	}
	return false, nil
}

// BaseAnimationWidget provides a base implementation for animation widgets.
// Embed this in widgets that need animation support.
type BaseAnimationWidget struct {
	BaseWidget
	overlay       *Overlay
	animBounds    []Rect
	needsAnimation bool
}

// SetOverlay sets the overlay this widget is registered with.
func (b *BaseAnimationWidget) SetOverlay(o *Overlay) {
	// Unregister from old overlay
	if b.overlay != nil {
		b.overlay.UnregisterAnimation(b)
	}
	b.overlay = o
}

// Overlay returns the overlay this widget is registered with.
func (b *BaseAnimationWidget) Overlay() *Overlay {
	return b.overlay
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

// FindOverlay walks up the widget tree to find the nearest Overlay.
func FindOverlay(w Widget) *Overlay {
	for w != nil {
		if o, ok := w.(*Overlay); ok {
			return o
		}
		w = w.Parent()
	}
	return nil
}
