package teapot

import tea "github.com/charmbracelet/bubbletea"

// View is the core interface for all UI components.
// Widgets form a tree structure where containers manage their children's layout.
type View interface {
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

	// Focus and input handling
	AcceptsFocus() bool // Returns true if the view is able to accept focus
	FocusNext() bool    // Moves focus to next focusable child; returns true if focus changed
	FocusPrev() bool    // Moves focus to previous focusable child; returns true if focus changed
	Focused() bool     // Returns true if this view currently has focus
	SetFocused(focused bool)
	HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd)
	HandleMouse(msg tea.MouseMsg) (handled bool, cmd tea.Cmd)

	// Tree structure
	Children() []View
	Parent() View
	SetParent(parent View)
}

// BaseView provides a default implementation of Widget.
// Embed this in concrete widget types to get sensible defaults.
type BaseView struct {
	name         string // View name (typically the struct type name)
	bounds       Rect
	constraints  Constraints
	focused      bool
	focusable    bool
	parent       View
	border       Border
	borderTitle  string
	borderFooter string
	dirty        bool // True if widget needs repainting
}

// NewBaseView creates a new base widget with the given z-order.
func NewBaseView() BaseView {
	return BaseView{
		constraints: DefaultConstraints(),
		focusable:   true,
		dirty:       true, // Widgets start dirty so they're rendered initially
	}
}

// Name returns the widget's name.
func (b *BaseView) Name() string {
	return b.name
}

// SetName sets the widget's name.
func (b *BaseView) SetName(name string) {
	b.name = name
}

// Constraints returns the widget's size constraints.
func (b *BaseView) Constraints() Constraints {
	return b.constraints
}

// SetConstraints updates the widget's size constraints.
func (b *BaseView) SetConstraints(c Constraints) {
	b.constraints = c
}

// SetBounds sets the widget's position and size.
func (b *BaseView) SetBounds(bounds Rect) {
	b.bounds = bounds
}

// Bounds returns the widget's current bounds.
func (b *BaseView) Bounds() Rect {
	return b.bounds
}

// Render is a no-op in the base widget.
func (b *BaseView) Render(buf *SubBuffer) {
	// No-op: override in concrete implementations
}

// AcceptsFocus returns whether this widget is able to accept focus.
// BaseView returns false by default; leaf views that can accept focus should override this.
func (b *BaseView) AcceptsFocus() bool {
	return false
}

// FocusNext moves focus to the next focusable child.
// For views that AcceptsFocus(), returns false (no children to focus).
// For views that don't AcceptsFocus(), panics (should not be called).
func (b *BaseView) FocusNext() bool {
	if !b.AcceptsFocus() {
		panic("FocusNext called on view that does not accept focus")
	}
	return false
}

// FocusPrev moves focus to the previous focusable child.
// For views that AcceptsFocus(), returns false (no children to focus).
// For views that don't AcceptsFocus(), panics (should not be called).
func (b *BaseView) FocusPrev() bool {
	if !b.AcceptsFocus() {
		panic("FocusPrev called on view that does not accept focus")
	}
	return false
}

// SetFocusable sets whether this widget can receive focus.
func (b *BaseView) SetFocusable(focusable bool) {
	b.focusable = focusable
}

// Focused returns whether this widget currently has focus.
func (b *BaseView) Focused() bool {
	return b.focused
}

// SetFocused sets the focus state.
func (b *BaseView) SetFocused(focused bool) {
	b.focused = focused
}

// HandleKey handles keyboard input.
func (b *BaseView) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	return false, nil
}

// HandleMouse handles mouse input.
func (b *BaseView) HandleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	return false, nil
}

// Children returns the widget's children (none for base widget).
func (b *BaseView) Children() []View {
	return nil
}

// Parent returns the widget's parent.
func (b *BaseView) Parent() View {
	return b.parent
}

// SetParent sets the widget's parent.
func (b *BaseView) SetParent(parent View) {
	b.parent = parent
}

// IsDirty returns true if the widget needs repainting.
func (b *BaseView) IsDirty() bool {
	return b.dirty
}

// Repaint marks this widget as needing repaint and propagates to parents.
func (b *BaseView) Repaint() {
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
func (b *BaseView) MightBeDirty() bool {
	return b.dirty
}

// Border returns the widget's border configuration.
func (b *BaseView) Border() Border {
	return b.border
}

// SetBorder sets the widget's border configuration.
func (b *BaseView) SetBorder(border Border) {
	b.border = border
}

// BorderTitle returns the widget's border title.
func (b *BaseView) BorderTitle() string {
	return b.borderTitle
}

// SetBorderTitle sets the widget's border title.
func (b *BaseView) SetBorderTitle(title string) {
	b.borderTitle = title
}

// BorderFooter returns the widget's border footer.
func (b *BaseView) BorderFooter() string {
	return b.borderFooter
}

// SetBorderFooter sets the widget's border footer.
func (b *BaseView) SetBorderFooter(footer string) {
	b.borderFooter = footer
}

// ContentBounds returns the bounds inside the border.
// If no border is set, returns the full bounds.
func (b *BaseView) ContentBounds() Rect {
	bounds := b.bounds
	border := b.border

	return Rect{
		Position{
			X: bounds.X + border.LeftWidth(),
			Y: bounds.Y + border.TopWidth(),
		},
		Size{
			Width:  bounds.Width - border.LeftWidth() - border.RightWidth(),
			Height: bounds.Height - border.TopWidth() - border.BottomWidth(),
		},
	}
}

// ContainerView extends BaseView with child management.
type ContainerView struct {
	BaseView
	children []View
}

// NewContainerView creates a new container widget.
func NewContainerView() ContainerView {
	return ContainerView{
		BaseView: NewBaseView(),
	}
}

// Children returns the container's children.
func (c *ContainerView) Children() []View {
	return c.children
}

// AddChild adds a child widget to this container.
func (c *ContainerView) AddChild(child View) {
	child.SetParent(c)
	c.children = append(c.children, child)
}

// RemoveChild removes a child widget from this container.
func (c *ContainerView) RemoveChild(child View) {
	for i, ch := range c.children {
		if ch == child {
			child.SetParent(nil)
			c.children = append(c.children[:i], c.children[i+1:]...)
			return
		}
	}
}

// ClearChildren removes all children from this container.
func (c *ContainerView) ClearChildren() {
	for _, child := range c.children {
		child.SetParent(nil)
	}
	c.children = nil
}

