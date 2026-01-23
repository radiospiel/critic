package teapot

import (
	"strings"

	"git.15b.it/eno/critic/simple-go/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Orientation specifies the direction of layout.
type Orientation int

const (
	Horizontal Orientation = iota
	Vertical
)

// BoxLayout is a container that arranges children in a line (horizontal or vertical).
// It distributes space based on constraints and stretch factors.
type BoxLayout struct {
	ContainerView
	orientation Orientation
	spacing     int
}

// NewVBox creates a vertical box layout.
func NewVBox(spacing int) *BoxLayout {
	box := &BoxLayout{
		ContainerView: NewContainerView(),
		orientation:   Vertical,
		spacing:       spacing,
	}
	box.SetFocusable(false)
	return box
}

// NewHBox creates a horizontal box layout.
func NewHBox(spacing int) *BoxLayout {
	box := &BoxLayout{
		ContainerView: NewContainerView(),
		orientation:   Horizontal,
		spacing:       spacing,
	}
	box.SetFocusable(false)
	return box
}

// SetBounds sets the bounds and performs layout on children.
func (b *BoxLayout) SetBounds(bounds Rect) {
	b.bounds = bounds
	b.layout()
}

// layout distributes space among children based on their constraints.
func (b *BoxLayout) layout() {
	if len(b.children) == 0 {
		return
	}

	availableWidth := b.bounds.Width
	availableHeight := b.bounds.Height

	// Calculate total spacing
	totalSpacing := b.spacing * (len(b.children) - 1)

	if b.orientation == Horizontal {
		b.layoutHorizontal(availableWidth-totalSpacing, availableHeight)
	} else {
		b.layoutVertical(availableWidth, availableHeight-totalSpacing)
	}
}

func (b *BoxLayout) layoutHorizontal(availableWidth, availableHeight int) {
	// First pass: calculate minimum and stretch totals
	totalMinWidth := 0
	totalStretch := 0
	for _, child := range b.children {
		c := child.Constraints()
		totalMinWidth += c.MinWidth
		totalStretch += c.HorizontalStretch
	}

	// Calculate extra space to distribute
	extraSpace := availableWidth - totalMinWidth
	if extraSpace < 0 {
		extraSpace = 0
	}

	// Second pass: assign sizes
	x := b.bounds.X
	for i, child := range b.children {
		c := child.Constraints()

		// Calculate width
		width := c.MinWidth
		if totalStretch > 0 && c.HorizontalStretch > 0 {
			width += (extraSpace * c.HorizontalStretch) / totalStretch
		} else if c.PreferredWidth > 0 && c.PreferredWidth > width {
			// Use preferred if no stretch and space available
			width = min(c.PreferredWidth, availableWidth)
		}

		// Set bounds
		child.SetBounds(Rect{
			Position{
				X: x,
				Y: b.bounds.Y,
			},
			Size{
				Width:  width,
				Height: availableHeight,
			},
		})

		x += width
		if i < len(b.children)-1 {
			x += b.spacing
		}
	}
}

func (b *BoxLayout) layoutVertical(availableWidth, availableHeight int) {
	// First pass: calculate minimum and stretch totals
	totalMinHeight := 0
	totalStretch := 0
	for _, child := range b.children {
		c := child.Constraints()
		totalMinHeight += c.MinHeight
		totalStretch += c.VerticalStretch
	}

	// Calculate extra space to distribute
	extraSpace := availableHeight - totalMinHeight
	if extraSpace < 0 {
		extraSpace = 0
	}

	// Second pass: assign sizes
	y := b.bounds.Y
	for i, child := range b.children {
		c := child.Constraints()

		// Calculate height
		height := c.MinHeight
		if totalStretch > 0 && c.VerticalStretch > 0 {
			height += (extraSpace * c.VerticalStretch) / totalStretch
		} else if c.PreferredHeight > 0 && c.PreferredHeight > height {
			height = min(c.PreferredHeight, availableHeight)
		}

		// Set bounds
		child.SetBounds(Rect{
			Position{
				X: b.bounds.X,
				Y: y,
			},
			Size{
				Width:  availableWidth,
				Height: height,
			},
		})

		y += height
		if i < len(b.children)-1 {
			y += b.spacing
		}
	}
}

// Render renders all children.
func (b *BoxLayout) Render(buf *SubBuffer) {
	for _, child := range b.children {
		childBounds := child.Bounds()
		// Create a sub-buffer for the child, relative to our position
		relX := childBounds.X - b.bounds.X
		relY := childBounds.Y - b.bounds.Y
		childSub := NewSubBuffer(buf.parent, Rect{
			Position{
				X: buf.offset.X + relX,
				Y: buf.offset.Y + relY,
			},
			Size{
				Width:  childBounds.Width,
				Height: childBounds.Height,
			},
		})
		RenderWidget(child, childSub)
	}
}

// HandleKey routes key events to focused child.
func (b *BoxLayout) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	for _, child := range b.children {
		if child.HasFocus() {
			return child.HandleKey(msg)
		}
		// Check for focused descendants
		if handled, cmd := b.routeToFocusedDescendant(child, msg); handled {
			return handled, cmd
		}
	}
	return false, nil
}

func (b *BoxLayout) routeToFocusedDescendant(w View, msg tea.KeyMsg) (bool, tea.Cmd) {
	for _, child := range w.Children() {
		if child.HasFocus() {
			return child.HandleKey(msg)
		}
		if handled, cmd := b.routeToFocusedDescendant(child, msg); handled {
			return handled, cmd
		}
	}
	return false, nil
}

// Split is a container with two panes separated by a divider.
// It supports both fixed-size and proportional layouts.
type Split struct {
	ContainerView
	orientation  Orientation
	first        View
	second       View
	ratio        float64 // 0.0 to 1.0, proportion of space for first pane
	fixedSize    int     // If > 0, first pane has fixed size
	dividerWidth int     // Width of the divider (default 1)
	dividerStyle lipgloss.Style
}

// NewHSplit creates a horizontal split (left | right).
func NewHSplit(left, right View, ratio float64) *Split {
	s := &Split{
		ContainerView: NewContainerView(),
		orientation:   Horizontal,
		first:         left,
		second:        right,
		ratio:         ratio,
		dividerWidth:  1,
		dividerStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}
	s.SetFocusable(false)
	if left != nil {
		left.SetParent(s)
	}
	if right != nil {
		right.SetParent(s)
	}
	return s
}

// NewVSplit creates a vertical split (top / bottom).
func NewVSplit(top, bottom View, ratio float64) *Split {
	s := &Split{
		ContainerView: NewContainerView(),
		orientation:   Vertical,
		first:         top,
		second:        bottom,
		ratio:         ratio,
		dividerWidth:  1,
		dividerStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}
	s.SetFocusable(false)
	if top != nil {
		top.SetParent(s)
	}
	if bottom != nil {
		bottom.SetParent(s)
	}
	return s
}

// SetRatio sets the split ratio (0.0 to 1.0).
func (s *Split) SetRatio(ratio float64) {
	s.ratio = utils.Clamp(ratio, 0.0, 1.0)
	s.Repaint()
}

// SetFixedSize sets a fixed size for the first pane (0 to use ratio instead).
func (s *Split) SetFixedSize(size int) {
	s.fixedSize = size
	s.Repaint()
}

// SetDividerStyle sets the style for the divider line.
func (s *Split) SetDividerStyle(style lipgloss.Style) {
	s.dividerStyle = style
	s.Repaint()
}

// Children returns the split's children.
func (s *Split) Children() []View {
	var children []View
	if s.first != nil {
		children = append(children, s.first)
	}
	if s.second != nil {
		children = append(children, s.second)
	}
	return children
}

// SetBounds sets the bounds and lays out the two panes.
func (s *Split) SetBounds(bounds Rect) {
	s.bounds = bounds

	if s.orientation == Horizontal {
		s.layoutHorizontal()
	} else {
		s.layoutVertical()
	}
}

func (s *Split) layoutHorizontal() {
	availableWidth := s.bounds.Width - s.dividerWidth

	var firstWidth int
	if s.fixedSize > 0 {
		firstWidth = min(s.fixedSize, availableWidth)
	} else {
		firstWidth = int(float64(availableWidth) * s.ratio)
	}
	secondWidth := availableWidth - firstWidth

	if s.first != nil {
		s.first.SetBounds(Rect{
			Position{
				X: s.bounds.X,
				Y: s.bounds.Y,
			},
			Size{
				Width:  firstWidth,
				Height: s.bounds.Height,
			},
		})
	}

	if s.second != nil {
		s.second.SetBounds(Rect{
			Position{
				X: s.bounds.X + firstWidth + s.dividerWidth,
				Y: s.bounds.Y,
			},
			Size{
				Width:  secondWidth,
				Height: s.bounds.Height,
			},
		})
	}
}

func (s *Split) layoutVertical() {
	availableHeight := s.bounds.Height - s.dividerWidth

	var firstHeight int
	if s.fixedSize > 0 {
		firstHeight = min(s.fixedSize, availableHeight)
	} else {
		firstHeight = int(float64(availableHeight) * s.ratio)
	}
	secondHeight := availableHeight - firstHeight

	if s.first != nil {
		s.first.SetBounds(Rect{
			Position{
				X: s.bounds.X,
				Y: s.bounds.Y,
			},
			Size{
				Width:  s.bounds.Width,
				Height: firstHeight,
			},
		})
	}

	if s.second != nil {
		s.second.SetBounds(Rect{
			Position{
				X: s.bounds.X,
				Y: s.bounds.Y + firstHeight + s.dividerWidth,
			},
			Size{
				Width:  s.bounds.Width,
				Height: secondHeight,
			},
		})
	}
}

// Render renders both panes and the divider.
func (s *Split) Render(buf *SubBuffer) {
	// Render divider
	if s.orientation == Horizontal {
		dividerX := 0
		if s.first != nil {
			dividerX = s.first.Bounds().Width
		}
		for y := 0; y < buf.Height(); y++ {
			buf.SetString(dividerX, y, "│", s.dividerStyle)
		}
	} else {
		dividerY := 0
		if s.first != nil {
			dividerY = s.first.Bounds().Height
		}
		buf.SetString(0, dividerY, strings.Repeat("─", buf.Width()), s.dividerStyle)
	}

	// Render children
	if s.first != nil {
		firstBounds := s.first.Bounds()
		firstSub := NewSubBuffer(buf.parent, Rect{
			Position{
				X: buf.offset.X + firstBounds.X - s.bounds.X,
				Y: buf.offset.Y + firstBounds.Y - s.bounds.Y,
			},
			Size{
				Width:  firstBounds.Width,
				Height: firstBounds.Height,
			},
		})
		RenderWidget(s.first, firstSub)
	}

	if s.second != nil {
		secondBounds := s.second.Bounds()
		secondSub := NewSubBuffer(buf.parent, Rect{
			Position{
				X: buf.offset.X + secondBounds.X - s.bounds.X,
				Y: buf.offset.Y + secondBounds.Y - s.bounds.Y,
			},
			Size{
				Width:  secondBounds.Width,
				Height: secondBounds.Height,
			},
		})
		RenderWidget(s.second, secondSub)
	}
}

// HandleKey routes key events to the focused child.
func (s *Split) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Check first pane
	if s.first != nil {
		if s.first.HasFocus() {
			return s.first.HandleKey(msg)
		}
		if handled, cmd := s.routeToFocused(s.first, msg); handled {
			return handled, cmd
		}
	}

	// Check second pane
	if s.second != nil {
		if s.second.HasFocus() {
			return s.second.HandleKey(msg)
		}
		if handled, cmd := s.routeToFocused(s.second, msg); handled {
			return handled, cmd
		}
	}

	return false, nil
}

func (s *Split) routeToFocused(w View, msg tea.KeyMsg) (bool, tea.Cmd) {
	for _, child := range w.Children() {
		if child.HasFocus() {
			return child.HandleKey(msg)
		}
		if handled, cmd := s.routeToFocused(child, msg); handled {
			return handled, cmd
		}
	}
	return false, nil
}

// First returns the first pane widget.
func (s *Split) First() View {
	return s.first
}

// Second returns the second pane widget.
func (s *Split) Second() View {
	return s.second
}

// SetFirst sets the first pane widget.
func (s *Split) SetFirst(w View) {
	if s.first != nil {
		s.first.SetParent(nil)
	}
	s.first = w
	if w != nil {
		w.SetParent(s)
	}
	s.SetBounds(s.bounds) // Re-layout
	s.Repaint()
}

// SetSecond sets the second pane widget.
func (s *Split) SetSecond(w View) {
	if s.second != nil {
		s.second.SetParent(nil)
	}
	s.second = w
	if w != nil {
		w.SetParent(s)
	}
	s.SetBounds(s.bounds) // Re-layout
	s.Repaint()
}
