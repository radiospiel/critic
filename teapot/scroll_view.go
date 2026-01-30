package teapot

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// ScrollView is a container that scrolls multiple children vertically.
// Children are laid out one after another, and the view can be scrolled
// to show different portions of the content.
// Optionally supports fixed header and footer views that don't scroll.
// Uses charmbracelet/bubbles viewport for scrolling functionality.
type ScrollView struct {
	BaseView
	children      []View
	headerView    View // Optional fixed header (doesn't scroll)
	footerView    View // Optional fixed footer (doesn't scroll)
	viewport      viewport.Model
	contentHeight int    // Total height of all children
	contentCache  string // Cached pre-rendered content for viewport
	contentDirty  bool   // True if content needs re-rendering
}

// NewScrollView creates a new vertical scroll view with the given children.
// Children are laid out vertically, with the first child at the top.
func NewScrollView(children ...View) *ScrollView {
	sv := &ScrollView{
		BaseView:     NewBaseView(),
		children:     make([]View, 0, len(children)),
		viewport:     viewport.New(0, 0),
		contentDirty: true,
	}
	sv.SetFocusable(true)

	// Enable mouse wheel support
	sv.viewport.MouseWheelEnabled = true

	for _, child := range children {
		sv.AddChild(child)
	}

	return sv
}

// AddChild adds a child to the scroll view.
func (s *ScrollView) AddChild(child View) {
	child.SetParent(s)
	s.children = append(s.children, child)
	s.contentDirty = true
	s.layoutChildren()
}

// ClearChildren removes all children from the scroll view.
func (s *ScrollView) ClearChildren() {
	for _, child := range s.children {
		child.SetParent(nil)
	}
	s.children = s.children[:0]
	s.contentHeight = 0
	s.contentDirty = true
	s.contentCache = ""
	s.viewport.SetContent("")
	s.viewport.GotoTop()
}

// Children returns the scroll view's children.
func (s *ScrollView) Children() []View {
	return s.children
}

// SetHeaderView sets an optional fixed header view that doesn't scroll.
// The header is rendered at the top of the scroll view.
func (s *ScrollView) SetHeaderView(header View) {
	if s.headerView != nil {
		s.headerView.SetParent(nil)
	}
	s.headerView = header
	if header != nil {
		header.SetParent(s)
	}
	s.contentDirty = true
	s.layoutChildren()
}

// SetFooterView sets an optional fixed footer view that doesn't scroll.
// The footer is rendered at the bottom of the scroll view.
func (s *ScrollView) SetFooterView(footer View) {
	if s.footerView != nil {
		s.footerView.SetParent(nil)
	}
	s.footerView = footer
	if footer != nil {
		footer.SetParent(s)
	}
	s.contentDirty = true
	s.layoutChildren()
}

// HeaderView returns the header view, if any.
func (s *ScrollView) HeaderView() View {
	return s.headerView
}

// FooterView returns the footer view, if any.
func (s *ScrollView) FooterView() View {
	return s.footerView
}

// headerHeight returns the height of the header view, or 0 if none.
func (s *ScrollView) headerHeight() int {
	if s.headerView == nil {
		return 0
	}
	c := s.headerView.Constraints()
	if c.PreferredHeight > 0 {
		return c.PreferredHeight
	}
	if c.MinHeight > 0 {
		return c.MinHeight
	}
	return 1
}

// footerHeight returns the height of the footer view, or 0 if none.
func (s *ScrollView) footerHeight() int {
	if s.footerView == nil {
		return 0
	}
	c := s.footerView.Constraints()
	if c.PreferredHeight > 0 {
		return c.PreferredHeight
	}
	if c.MinHeight > 0 {
		return c.MinHeight
	}
	return 1
}

// scrollableHeight returns the height available for scrollable content.
func (s *ScrollView) scrollableHeight() int {
	return s.bounds.Height - s.headerHeight() - s.footerHeight()
}

// SetBounds sets the scroll view's bounds and lays out children.
func (s *ScrollView) SetBounds(bounds Rect) {
	s.BaseView.SetBounds(bounds)
	s.layoutChildren()
}

// layoutChildren lays out all children vertically and calculates total content height.
// Also lays out header and footer views if present, and updates viewport dimensions.
func (s *ScrollView) layoutChildren() {
	width := s.bounds.Width

	// Layout header if present
	hHeight := s.headerHeight()
	if s.headerView != nil {
		s.headerView.SetBounds(Rect{
			Position: Position{X: 0, Y: 0},
			Size:     Size{Width: width, Height: hHeight},
		})
	}

	// Layout footer if present
	fHeight := s.footerHeight()
	if s.footerView != nil {
		s.footerView.SetBounds(Rect{
			Position: Position{X: 0, Y: s.bounds.Height - fHeight},
			Size:     Size{Width: width, Height: fHeight},
		})
	}

	// Update viewport dimensions for the scrollable area
	scrollHeight := s.scrollableHeight()
	if scrollHeight < 0 {
		scrollHeight = 0
	}
	s.viewport.Width = width
	s.viewport.Height = scrollHeight

	// Layout scrollable children
	if len(s.children) == 0 {
		s.contentHeight = 0
		s.contentDirty = true
		return
	}

	y := 0
	for _, child := range s.children {
		constraints := child.Constraints()

		// Determine child height
		height := constraints.PreferredHeight
		if height == 0 {
			height = constraints.MinHeight
		}
		if height == 0 {
			height = 1 // Default minimum
		}

		child.SetBounds(Rect{
			Position: Position{X: 0, Y: y},
			Size:     Size{Width: width, Height: height},
		})

		y += height
	}

	s.contentHeight = y
	s.contentDirty = true
}

// ScrollOffset returns the current scroll offset.
func (s *ScrollView) ScrollOffset() int {
	return s.viewport.YOffset
}

// SetScrollOffset sets the scroll offset.
func (s *ScrollView) SetScrollOffset(offset int) {
	s.viewport.SetYOffset(offset)
}

// ScrollToTop scrolls to the top of the content.
func (s *ScrollView) ScrollToTop() {
	s.viewport.GotoTop()
}

// ScrollToBottom scrolls to the bottom of the content.
func (s *ScrollView) ScrollToBottom() {
	s.viewport.GotoBottom()
}

// Render renders the visible portion of the scroll view's content.
func (s *ScrollView) Render(buf *SubBuffer) {
	viewWidth := buf.Width()
	hHeight := s.headerHeight()
	fHeight := s.footerHeight()
	scrollHeight := s.scrollableHeight()

	// Render header (fixed at top)
	if s.headerView != nil && hHeight > 0 {
		headerSub := NewSubBuffer(buf.parent, Rect{
			Position: Position{
				X: buf.offset.X,
				Y: buf.offset.Y,
			},
			Size: Size{
				Width:  viewWidth,
				Height: hHeight,
			},
		})
		RenderWidget(s.headerView, headerSub)
	}

	// Render footer (fixed at bottom)
	if s.footerView != nil && fHeight > 0 {
		footerSub := NewSubBuffer(buf.parent, Rect{
			Position: Position{
				X: buf.offset.X,
				Y: buf.offset.Y + buf.Height() - fHeight,
			},
			Size: Size{
				Width:  viewWidth,
				Height: fHeight,
			},
		})
		RenderWidget(s.footerView, footerSub)
	}

	// Render scrollable children in the middle area
	if len(s.children) == 0 || scrollHeight <= 0 {
		return
	}

	// Update viewport content if dirty
	if s.contentDirty {
		s.updateViewportContent()
	}

	// Get the visible content from viewport and render it
	viewportContent := s.viewport.View()
	lines := strings.Split(viewportContent, "\n")

	for y, line := range lines {
		if y >= scrollHeight {
			break
		}
		// Parse the ANSI line back to cells
		cells := ParseANSILine(line)

		// Pad to full width if needed
		if len(cells) < viewWidth {
			padding := make([]Cell, viewWidth-len(cells))
			for i := range padding {
				padding[i] = EmptyCell
			}
			cells = append(cells, padding...)
		} else if len(cells) > viewWidth {
			cells = cells[:viewWidth]
		}

		// Write to buffer at the correct Y position (offset by header)
		buf.SetCells(0, hHeight+y, cells)
	}
}

// updateViewportContent pre-renders all children to a string and updates the viewport.
func (s *ScrollView) updateViewportContent() {
	if len(s.children) == 0 {
		s.contentCache = ""
		s.viewport.SetContent("")
		s.contentDirty = false
		return
	}

	width := s.bounds.Width
	if width <= 0 {
		s.contentCache = ""
		s.viewport.SetContent("")
		s.contentDirty = false
		return
	}

	// Create a buffer for all children content
	contentBuf := NewBuffer(width, s.contentHeight)

	// Render each child to the content buffer
	for _, child := range s.children {
		childBounds := child.Bounds()
		childSub := NewSubBuffer(contentBuf, Rect{
			Position: Position{X: 0, Y: childBounds.Y},
			Size:     Size{Width: width, Height: childBounds.Height},
		})
		RenderWidget(child, childSub)
	}

	// Convert buffer to string for viewport
	s.contentCache = contentBuf.RenderToString()
	s.viewport.SetContent(s.contentCache)
	s.contentDirty = false
}

// HandleKey handles keyboard input for scrolling.
// Uses viewport's built-in scrolling methods.
func (s *ScrollView) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "up":
		if !s.viewport.AtTop() {
			s.viewport.LineUp(1)
			return true, nil
		}
	case "down":
		if !s.viewport.AtBottom() {
			s.viewport.LineDown(1)
			return true, nil
		}
	case "ctrl+up", "pgup":
		s.viewport.HalfViewUp()
		return true, nil
	case "ctrl+down", "pgdown", " ": // space for page down
		s.viewport.HalfViewDown()
		return true, nil
	case "home":
		s.viewport.GotoTop()
		return true, nil
	case "end":
		s.viewport.GotoBottom()
		return true, nil
	}

	return false, nil
}

// HandleMouse handles mouse input for scrolling.
// Passes mouse wheel events to the viewport for scroll handling.
func (s *ScrollView) HandleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	// Only handle mouse wheel events within the scrollable area
	hHeight := s.headerHeight()
	fHeight := s.footerHeight()

	// Check if mouse is in the scrollable area
	if msg.Y < hHeight || msg.Y >= s.bounds.Height-fHeight {
		return false, nil
	}

	// Handle mouse wheel scrolling
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		s.viewport.LineUp(3)
		return true, nil
	case tea.MouseButtonWheelDown:
		s.viewport.LineDown(3)
		return true, nil
	}

	return false, nil
}

// AcceptsFocus returns true as the scroll view can receive focus for keyboard scrolling.
func (s *ScrollView) AcceptsFocus() bool {
	return true
}

// FocusNext is a no-op for scroll view as it has no focusable children.
func (s *ScrollView) FocusNext() bool {
	return false
}

// FocusPrev is a no-op for scroll view as it has no focusable children.
func (s *ScrollView) FocusPrev() bool {
	return false
}
