// Package widget provides a composable widget system for terminal UIs.
// Inspired by Qt's layout system, widgets form a tree where parent containers
// manage the layout of their children, and focus/input events flow through
// the hierarchy.
package widget

// Rect represents a rectangular area with position and size.
type Rect struct {
	X, Y          int
	Width, Height int
}

// NewRect creates a new rectangle.
func NewRect(x, y, width, height int) Rect {
	return Rect{X: x, Y: y, Width: width, Height: height}
}

// Contains returns true if the point (px, py) is inside the rectangle.
func (r Rect) Contains(px, py int) bool {
	return px >= r.X && px < r.X+r.Width && py >= r.Y && py < r.Y+r.Height
}

// Intersect returns the intersection of two rectangles.
// If they don't intersect, returns a zero-sized rectangle.
func (r Rect) Intersect(other Rect) Rect {
	x1 := max(r.X, other.X)
	y1 := max(r.Y, other.Y)
	x2 := min(r.X+r.Width, other.X+other.Width)
	y2 := min(r.Y+r.Height, other.Y+other.Height)

	if x2 <= x1 || y2 <= y1 {
		return Rect{}
	}
	return Rect{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

// IsEmpty returns true if the rectangle has no area.
func (r Rect) IsEmpty() bool {
	return r.Width <= 0 || r.Height <= 0
}

// Inset returns a new rectangle shrunk by the given margins.
func (r Rect) Inset(top, right, bottom, left int) Rect {
	return Rect{
		X:      r.X + left,
		Y:      r.Y + top,
		Width:  max(0, r.Width-left-right),
		Height: max(0, r.Height-top-bottom),
	}
}

// Size returns just the width and height as a Size.
func (r Rect) Size() Size {
	return Size{Width: r.Width, Height: r.Height}
}

// Size represents dimensions without position.
type Size struct {
	Width, Height int
}

// Constraints represents layout constraints for a widget.
// Zero values mean "no constraint".
type Constraints struct {
	MinWidth, MinHeight   int
	MaxWidth, MaxHeight   int
	PreferredWidth        int
	PreferredHeight       int
	HorizontalStretch     int // Stretch factor for flexible sizing (0 = fixed)
	VerticalStretch       int
}

// DefaultConstraints returns constraints with no limits.
func DefaultConstraints() Constraints {
	return Constraints{
		MaxWidth:  999999,
		MaxHeight: 999999,
	}
}

// WithMinSize returns constraints with minimum size set.
func (c Constraints) WithMinSize(w, h int) Constraints {
	c.MinWidth = w
	c.MinHeight = h
	return c
}

// WithPreferredSize returns constraints with preferred size set.
func (c Constraints) WithPreferredSize(w, h int) Constraints {
	c.PreferredWidth = w
	c.PreferredHeight = h
	return c
}

// WithStretch returns constraints with stretch factors set.
func (c Constraints) WithStretch(h, v int) Constraints {
	c.HorizontalStretch = h
	c.VerticalStretch = v
	return c
}

// EffectivePreferredWidth returns the preferred width, falling back to min.
func (c Constraints) EffectivePreferredWidth() int {
	if c.PreferredWidth > 0 {
		return c.PreferredWidth
	}
	return c.MinWidth
}

// EffectivePreferredHeight returns the preferred height, falling back to min.
func (c Constraints) EffectivePreferredHeight() int {
	if c.PreferredHeight > 0 {
		return c.PreferredHeight
	}
	return c.MinHeight
}
