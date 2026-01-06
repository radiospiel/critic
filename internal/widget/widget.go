package widget

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Widget is the core interface for all UI components.
// Widgets form a tree structure where containers manage their children's layout.
type Widget interface {
	// Layout and sizing
	Constraints() Constraints
	SetBounds(bounds Rect)
	Bounds() Rect

	// Rendering
	Render(buf *SubBuffer)

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

// BaseWidget provides a default implementation of Widget.
// Embed this in concrete widget types to get sensible defaults.
type BaseWidget struct {
	bounds      Rect
	constraints Constraints
	focused     bool
	focusable   bool
	parent      Widget
}

// NewBaseWidget creates a new base widget with default settings.
func NewBaseWidget() BaseWidget {
	return BaseWidget{
		constraints: DefaultConstraints(),
		focusable:   true,
	}
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

// ContainerWidget extends BaseWidget with child management.
type ContainerWidget struct {
	BaseWidget
	children []Widget
}

// NewContainerWidget creates a new container widget.
func NewContainerWidget() ContainerWidget {
	return ContainerWidget{
		BaseWidget: NewBaseWidget(),
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
	root        Widget
	focused     Widget
	focusChain  []Widget
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
