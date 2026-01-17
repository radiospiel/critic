package teapot

import (
	"git.15b.it/eno/critic/simple-go/logger"
	tea "github.com/charmbracelet/bubbletea"
)

// Z-order constants for rendering layers
const (
	// ZOrderDefault is the default z-order for normal widgets
	ZOrderDefault = 1
	// ZOrderAnimation is the z-order for animation overlays
	ZOrderAnimation = 100
)

// Widget is the core interface for all UI components.
// Widgets form a tree structure where containers manage their children's layout.
type Widget interface {
	// Identity
	Name() string // Returns the widget's name (typically the struct type name)

	// Layout and sizing
	Constraints() Constraints
	SetBounds(bounds Rect)
	Bounds() Rect

	// Borders
	Border() Border
	SetBorder(border Border)
	BorderTitle() string
	SetBorderTitle(title string)
	BorderFooter() string
	SetBorderFooter(footer string)
	ContentBounds() Rect // Returns bounds inside the border

	// Rendering
	Render(buf *SubBuffer)

	// Dirty tracking
	IsDirty() bool      // Returns true if widget needs repainting
	Repaint()           // Marks widget as needing repaint (propagates to parents)
	MightBeDirty() bool // Returns true if widget might need repainting (animated widgets override to return true)
	ZOrder() int        // Returns the z-order for rendering layers

	// Focus and input handling
	Focusable() bool
	Focused() bool
	SetFocused(focused bool)
	HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd)
	HandleMouse(msg tea.MouseMsg) (handled bool, cmd tea.Cmd)

	// Tree structure
	Children() []Widget
	Parent() Widget
	SetParent(parent Widget)
}

// cacheableWidget is an internal interface used by the Compositor to manage
// per-widget render caching. This interface is intentionally not part of the
// public Widget interface because caching is an implementation detail of the
// compositor's rendering strategy. Widgets that embed BaseWidget automatically
// implement this interface.
type cacheableWidget interface {
	CachedView() *Buffer
	SetCachedView(*Buffer)
}

// BaseWidget provides a default implementation of Widget.
// Embed this in concrete widget types to get sensible defaults.
type BaseWidget struct {
	name         string // Widget name (typically the struct type name)
	bounds       Rect
	constraints  Constraints
	focused      bool
	focusable    bool
	parent       Widget
	border       Border
	borderTitle  string
	borderFooter string
	dirty        bool    // True if widget needs repainting
	zOrder       int     // Z-order for rendering layers (default: ZOrderDefault)
	cachedView   *Buffer // Cached rendered view (owned by widget, not compositor)
}

// NewBaseWidget creates a new base widget with the given z-order.
func NewBaseWidget(zOrder int) BaseWidget {
	return BaseWidget{
		constraints: DefaultConstraints(),
		focusable:   true,
		dirty:       true, // Widgets start dirty so they're rendered initially
		zOrder:      zOrder,
	}
}

// Name returns the widget's name.
func (b *BaseWidget) Name() string {
	return b.name
}

// SetName sets the widget's name.
func (b *BaseWidget) SetName(name string) {
	b.name = name
}

// Constraints returns the widget's size constraints.
func (b *BaseWidget) Constraints() Constraints {
	return b.constraints
}

// SetConstraints updates the widget's size constraints.
func (b *BaseWidget) SetConstraints(c Constraints) {
	b.constraints = c
}

// SetBounds sets the widget's position and size.
func (b *BaseWidget) SetBounds(bounds Rect) {
	b.bounds = bounds
}

// Bounds returns the widget's current bounds.
func (b *BaseWidget) Bounds() Rect {
	return b.bounds
}

// Render is a no-op in the base widget.
func (b *BaseWidget) Render(buf *SubBuffer) {
	// No-op: override in concrete implementations
}

// Focusable returns whether this widget can receive focus.
func (b *BaseWidget) Focusable() bool {
	return b.focusable
}

// SetFocusable sets whether this widget can receive focus.
func (b *BaseWidget) SetFocusable(focusable bool) {
	b.focusable = focusable
}

// Focused returns whether this widget currently has focus.
func (b *BaseWidget) Focused() bool {
	return b.focused
}

// SetFocused sets the focus state.
func (b *BaseWidget) SetFocused(focused bool) {
	b.focused = focused
}

// HandleKey handles keyboard input.
func (b *BaseWidget) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	return false, nil
}

// HandleMouse handles mouse input.
func (b *BaseWidget) HandleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	return false, nil
}

// Children returns the widget's children (none for base widget).
func (b *BaseWidget) Children() []Widget {
	return nil
}

// Parent returns the widget's parent.
func (b *BaseWidget) Parent() Widget {
	return b.parent
}

// SetParent sets the widget's parent.
func (b *BaseWidget) SetParent(parent Widget) {
	b.parent = parent
}

// IsDirty returns true if the widget needs repainting.
func (b *BaseWidget) IsDirty() bool {
	return b.dirty
}

// Repaint marks this widget as needing repaint and propagates to parents.
func (b *BaseWidget) Repaint() {
	if b.dirty {
		return // Already dirty, no need to propagate
	}
	b.dirty = true
	if b.parent != nil {
		b.parent.Repaint()
	}
}

// MightBeDirty returns true if the widget might need repainting.
// For normal widgets, this returns true only if dirty.
// Animated widgets should override this to always return true.
func (b *BaseWidget) MightBeDirty() bool {
	return b.dirty
}

// ZOrder returns the widget's z-order for rendering.
func (b *BaseWidget) ZOrder() int {
	return b.zOrder
}

// CachedView returns the cached rendered view, or nil if not cached.
func (b *BaseWidget) CachedView() *Buffer {
	return b.cachedView
}

// SetCachedView sets the cached rendered view and clears the dirty flag.
// This should be called after rendering to cache.
func (b *BaseWidget) SetCachedView(buf *Buffer) {
	b.cachedView = buf
	if buf != nil {
		b.dirty = false // Clear dirty after caching the view
	}
}

// Border returns the widget's border configuration.
func (b *BaseWidget) Border() Border {
	return b.border
}

// SetBorder sets the widget's border configuration.
func (b *BaseWidget) SetBorder(border Border) {
	b.border = border
}

// BorderTitle returns the widget's border title.
func (b *BaseWidget) BorderTitle() string {
	return b.borderTitle
}

// SetBorderTitle sets the widget's border title.
func (b *BaseWidget) SetBorderTitle(title string) {
	b.borderTitle = title
}

// BorderFooter returns the widget's border footer.
func (b *BaseWidget) BorderFooter() string {
	return b.borderFooter
}

// SetBorderFooter sets the widget's border footer.
func (b *BaseWidget) SetBorderFooter(footer string) {
	b.borderFooter = footer
}

// ContentBounds returns the bounds inside the border.
// If no border is set, returns the full bounds.
func (b *BaseWidget) ContentBounds() Rect {
	bounds := b.bounds
	border := b.border

	return Rect{
		X:      bounds.X + border.LeftWidth(),
		Y:      bounds.Y + border.TopWidth(),
		Width:  bounds.Width - border.LeftWidth() - border.RightWidth(),
		Height: bounds.Height - border.TopWidth() - border.BottomWidth(),
	}
}

// ContainerWidget extends BaseWidget with child management.
type ContainerWidget struct {
	BaseWidget
	children []Widget
}

// NewContainerWidget creates a new container widget.
func NewContainerWidget() ContainerWidget {
	return ContainerWidget{
		BaseWidget: NewBaseWidget(ZOrderDefault),
	}
}

// Children returns the container's children.
func (c *ContainerWidget) Children() []Widget {
	return c.children
}

// AddChild adds a child widget to this container.
func (c *ContainerWidget) AddChild(child Widget) {
	child.SetParent(c)
	c.children = append(c.children, child)
}

// RemoveChild removes a child widget from this container.
func (c *ContainerWidget) RemoveChild(child Widget) {
	for i, ch := range c.children {
		if ch == child {
			child.SetParent(nil)
			c.children = append(c.children[:i], c.children[i+1:]...)
			return
		}
	}
}

// ClearChildren removes all children from this container.
func (c *ContainerWidget) ClearChildren() {
	for _, child := range c.children {
		child.SetParent(nil)
	}
	c.children = nil
}

// FocusManager handles focus traversal within a widget tree.
type FocusManager struct {
	root       Widget
	focused    Widget
	focusChain []Widget
}

// NewFocusManager creates a new focus manager for the given widget tree.
func NewFocusManager(root Widget) *FocusManager {
	fm := &FocusManager{root: root}
	fm.rebuildFocusChain()
	return fm
}

// Focused returns the currently focused widget.
func (fm *FocusManager) Focused() Widget {
	return fm.focused
}

// SetFocused sets focus to the given widget.
func (fm *FocusManager) SetFocused(w Widget) {
	if fm.focused != nil {
		fm.focused.SetFocused(false)
	}
	fm.focused = w
	if w != nil {
		w.SetFocused(true)
	}
}

// FocusNext moves focus to the next focusable widget.
func (fm *FocusManager) FocusNext() {
	if len(fm.focusChain) == 0 {
		return
	}

	currentIdx := -1
	for i, w := range fm.focusChain {
		if w == fm.focused {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + 1) % len(fm.focusChain)
	fm.SetFocused(fm.focusChain[nextIdx])
}

// FocusPrev moves focus to the previous focusable widget.
func (fm *FocusManager) FocusPrev() {
	if len(fm.focusChain) == 0 {
		return
	}

	currentIdx := -1
	for i, w := range fm.focusChain {
		if w == fm.focused {
			currentIdx = i
			break
		}
	}

	prevIdx := currentIdx - 1
	if prevIdx < 0 {
		prevIdx = len(fm.focusChain) - 1
	}
	fm.SetFocused(fm.focusChain[prevIdx])
}

// RebuildFocusChain rebuilds the list of focusable widgets.
// Call this after adding/removing widgets.
func (fm *FocusManager) RebuildFocusChain() {
	fm.rebuildFocusChain()
}

func (fm *FocusManager) rebuildFocusChain() {
	fm.focusChain = nil
	fm.collectFocusable(fm.root)
}

func (fm *FocusManager) collectFocusable(w Widget) {
	if w == nil {
		return
	}

	if w.Focusable() {
		logger.Info("*** collectFocusable")
		fm.focusChain = append(fm.focusChain, w)
	}

	for _, child := range w.Children() {
		fm.collectFocusable(child)
	}
}

// HandleKey routes a key event through the focus system.
// Returns true if the event was handled.
func (fm *FocusManager) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Tab/Shift+Tab for focus navigation
	switch msg.String() {
	case "tab":
		fm.FocusNext()
		return true, nil
	case "shift+tab":
		fm.FocusPrev()
		return true, nil
	}

	// Route to focused widget
	if fm.focused != nil {
		return fm.focused.HandleKey(msg)
	}

	return false, nil
}
