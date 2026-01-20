package teapot

import tea "github.com/charmbracelet/bubbletea"

// ScrollView is a container that scrolls multiple children vertically.
// Children are laid out one after another, and the view can be scrolled
// to show different portions of the content.
// Optionally supports fixed header and footer views that don't scroll.
// Scrolling: up/down (line), ctrl+up/ctrl+down/space (page), home/end (bounds).
type ScrollView struct {
	BaseView
	children      []View
	headerView    View // Optional fixed header (doesn't scroll)
	footerView    View // Optional fixed footer (doesn't scroll)
	scrollOffset  int
	contentHeight int // Total height of all children
}

// NewScrollView creates a new vertical scroll view with the given children.
// Children are laid out vertically, with the first child at the top.
func NewScrollView(children ...View) *ScrollView {
	sv := &ScrollView{
		BaseView: NewBaseView(),
		children: make([]View, 0, len(children)),
	}
	sv.SetFocusable(true)

	for _, child := range children {
		sv.AddChild(child)
	}

	return sv
}

// AddChild adds a child to the scroll view.
func (s *ScrollView) AddChild(child View) {
	child.SetParent(s)
	s.children = append(s.children, child)
	s.layoutChildren()
}

// ClearChildren removes all children from the scroll view.
func (s *ScrollView) ClearChildren() {
	for _, child := range s.children {
		child.SetParent(nil)
	}
	s.children = s.children[:0]
	s.scrollOffset = 0
	s.contentHeight = 0
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
// Also lays out header and footer views if present.
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

	// Layout scrollable children
	if len(s.children) == 0 {
		s.contentHeight = 0
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
	s.clampScrollOffset()
}

// clampScrollOffset ensures scroll offset is within valid bounds.
func (s *ScrollView) clampScrollOffset() {
	maxScroll := s.maxScrollOffset()
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
	if s.scrollOffset > maxScroll {
		s.scrollOffset = maxScroll
	}
}

// maxScrollOffset returns the maximum valid scroll offset.
func (s *ScrollView) maxScrollOffset() int {
	maxScroll := s.contentHeight - s.scrollableHeight()
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

// ScrollOffset returns the current scroll offset.
func (s *ScrollView) ScrollOffset() int {
	return s.scrollOffset
}

// SetScrollOffset sets the scroll offset.
func (s *ScrollView) SetScrollOffset(offset int) {
	s.scrollOffset = offset
	s.clampScrollOffset()
}

// ScrollToTop scrolls to the top of the content.
func (s *ScrollView) ScrollToTop() {
	s.scrollOffset = 0
}

// ScrollToBottom scrolls to the bottom of the content.
func (s *ScrollView) ScrollToBottom() {
	s.scrollOffset = s.maxScrollOffset()
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

	// Render each child that is at least partially visible
	for _, child := range s.children {
		childBounds := child.Bounds()

		// Calculate child's position relative to scroll offset
		childTop := childBounds.Y - s.scrollOffset
		childBottom := childTop + childBounds.Height

		// Skip if completely above or below the scrollable area
		if childBottom <= 0 || childTop >= scrollHeight {
			continue
		}

		// Calculate the visible portion of the child
		visibleTop := max(0, childTop)
		visibleBottom := min(scrollHeight, childBottom)
		visibleHeight := visibleBottom - visibleTop

		// Calculate offset within the child (for partial visibility at top)
		childYOffset := 0
		if childTop < 0 {
			childYOffset = -childTop
		}

		// Create a sub-buffer for the visible portion (offset by header height)
		childSub := NewSubBuffer(buf.parent, Rect{
			Position: Position{
				X: buf.offset.X,
				Y: buf.offset.Y + hHeight + visibleTop,
			},
			Size: Size{
				Width:  viewWidth,
				Height: visibleHeight,
			},
		})

		// Render the child with adjusted bounds for partial visibility
		if childYOffset > 0 || visibleHeight < childBounds.Height {
			// Child is partially visible - we need to render it offset
			s.renderChildPartial(child, childSub, childYOffset, visibleHeight)
		} else {
			// Child is fully visible
			RenderWidget(child, childSub)
		}
	}
}

// renderChildPartial renders a portion of a child starting at yOffset.
func (s *ScrollView) renderChildPartial(child View, buf *SubBuffer, yOffset, height int) {
	// Create a temporary buffer for the full child render
	childBounds := child.Bounds()
	tempBuf := NewBuffer(childBounds.Width, childBounds.Height)
	tempSubBuf := NewSubBuffer(tempBuf, Rect{
		Position: Position{X: 0, Y: 0},
		Size:     Size{Width: childBounds.Width, Height: childBounds.Height},
	})

	// Render child to temp buffer
	RenderWidget(child, tempSubBuf)

	// Copy the visible portion to the output buffer
	width := min(buf.Width(), tempBuf.Width())
	for y := 0; y < height; y++ {
		srcY := yOffset + y
		if srcY >= tempBuf.Height() {
			break
		}
		// Get the row from temp buffer and set it in the output buffer
		rowCells := make([]Cell, width)
		for x := 0; x < width; x++ {
			rowCells[x] = tempBuf.GetCell(x, srcY)
		}
		buf.SetCells(0, y, rowCells)
	}
}

// HandleKey handles keyboard input for scrolling.
func (s *ScrollView) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	viewHeight := s.scrollableHeight()
	pageSize := viewHeight - 3
	if pageSize < 1 {
		pageSize = 1
	}

	switch msg.String() {
	case "up":
		if s.scrollOffset > 0 {
			s.scrollOffset--
			return true, nil
		}
	case "down":
		if s.scrollOffset < s.maxScrollOffset() {
			s.scrollOffset++
			return true, nil
		}
	case "ctrl+up":
		s.scrollOffset -= pageSize
		s.clampScrollOffset()
		return true, nil
	case "ctrl+down", " ": // space for page down
		s.scrollOffset += pageSize
		s.clampScrollOffset()
		return true, nil
	case "home":
		s.ScrollToTop()
		return true, nil
	case "end":
		s.ScrollToBottom()
		return true, nil
	}

	return false, nil
}

// Focusable returns true as the scroll view can receive focus for keyboard scrolling.
func (s *ScrollView) Focusable() bool {
	return true
}
