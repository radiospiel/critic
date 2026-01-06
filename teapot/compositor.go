package teapot

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Compositor manages the root widget tree and orchestrates rendering.
// It owns the screen buffer and handles the render loop.
type Compositor struct {
	root          Widget
	width, height int
	buffer        *Buffer
	prevBuffer    *Buffer
	focusManager  *FocusManager
	dirty         bool // True if a re-render is needed
}

// NewCompositor creates a new compositor with the given root widget.
func NewCompositor(root Widget) *Compositor {
	c := &Compositor{
		root:  root,
		dirty: true,
	}
	if root != nil {
		c.focusManager = NewFocusManager(root)
	}
	return c
}

// SetRoot sets the root widget.
func (c *Compositor) SetRoot(root Widget) {
	c.root = root
	if root != nil {
		root.SetBounds(Rect{X: 0, Y: 0, Width: c.width, Height: c.height})
		c.focusManager = NewFocusManager(root)
	} else {
		c.focusManager = nil
	}
	c.dirty = true
}

// Root returns the root widget.
func (c *Compositor) Root() Widget {
	return c.root
}

// Resize handles terminal resize events.
func (c *Compositor) Resize(width, height int) {
	c.width = width
	c.height = height

	// Reallocate buffers
	c.buffer = NewBuffer(width, height)
	c.prevBuffer = nil // Force full redraw

	// Propagate size to root
	if c.root != nil {
		c.root.SetBounds(Rect{X: 0, Y: 0, Width: width, Height: height})
	}

	c.dirty = true
}

// Size returns the current screen size.
func (c *Compositor) Size() (width, height int) {
	return c.width, c.height
}

// MarkDirty marks the compositor as needing a re-render.
func (c *Compositor) MarkDirty() {
	c.dirty = true
}

// IsDirty returns true if the compositor needs a re-render.
func (c *Compositor) IsDirty() bool {
	return c.dirty
}

// Render renders the widget tree to the buffer and returns the string output.
// This implements differential rendering - only changed cells are updated.
func (c *Compositor) Render() string {
	if c.buffer == nil || c.root == nil {
		return ""
	}

	// Clear and render
	c.buffer.Clear()
	sub := c.buffer.Sub(c.buffer.Bounds())
	c.root.Render(sub)

	// TODO: Implement true differential rendering by comparing with prevBuffer
	// For now, we do full renders but the infrastructure is in place

	// Save for next comparison
	c.prevBuffer = c.buffer.Clone()
	c.dirty = false

	return c.buffer.String()
}

// View returns the rendered output (for BubbleTea compatibility).
func (c *Compositor) View() string {
	return c.Render()
}

// HandleKey routes key events through the focus manager.
func (c *Compositor) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if c.focusManager != nil {
		return c.focusManager.HandleKey(msg)
	}
	return false, nil
}

// HandleMouse routes mouse events to the widget under the cursor.
func (c *Compositor) HandleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	if c.root == nil {
		return false, nil
	}
	return c.routeMouseEvent(c.root, msg)
}

func (c *Compositor) routeMouseEvent(w Widget, msg tea.MouseMsg) (bool, tea.Cmd) {
	bounds := w.Bounds()
	if !bounds.Contains(msg.X, msg.Y) {
		return false, nil
	}

	// Check children first (reverse order for proper z-ordering)
	children := w.Children()
	for i := len(children) - 1; i >= 0; i-- {
		if handled, cmd := c.routeMouseEvent(children[i], msg); handled {
			return handled, cmd
		}
	}

	// Then the widget itself
	return w.HandleMouse(msg)
}

// FocusManager returns the focus manager.
func (c *Compositor) FocusManager() *FocusManager {
	return c.focusManager
}

// SetFocused sets focus to a specific widget.
func (c *Compositor) SetFocused(w Widget) {
	if c.focusManager != nil {
		c.focusManager.SetFocused(w)
	}
}

// Focused returns the currently focused widget.
func (c *Compositor) Focused() Widget {
	if c.focusManager != nil {
		return c.focusManager.Focused()
	}
	return nil
}

// RebuildFocusChain rebuilds the focus chain after widget tree changes.
func (c *Compositor) RebuildFocusChain() {
	if c.focusManager != nil {
		c.focusManager.RebuildFocusChain()
	}
}

// Update handles BubbleTea messages and returns commands.
// This is the main integration point with BubbleTea.
func (c *Compositor) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.Resize(msg.Width, msg.Height)
		return nil

	case tea.KeyMsg:
		_, cmd := c.HandleKey(msg)
		c.dirty = true
		return cmd

	case tea.MouseMsg:
		_, cmd := c.HandleMouse(msg)
		c.dirty = true
		return cmd
	}

	return nil
}

// CompositorModel wraps a Compositor as a full BubbleTea Model.
// Use this when you want the compositor to be the entire application.
type CompositorModel struct {
	compositor *Compositor
}

// NewCompositorModel creates a new BubbleTea model wrapping a compositor.
func NewCompositorModel(root Widget) CompositorModel {
	return CompositorModel{
		compositor: NewCompositor(root),
	}
}

// Compositor returns the underlying compositor.
func (m CompositorModel) Compositor() *Compositor {
	return m.compositor
}

// Init implements tea.Model.
func (m CompositorModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m CompositorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	cmd := m.compositor.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m CompositorModel) View() string {
	return m.compositor.View()
}
