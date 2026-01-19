package teapot

import (
	"github.com/charmbracelet/lipgloss"
)

// SubBuffer returns a view into a rectangular region of this buffer.
// Writes to the sub-buffer affect the original.
type SubBuffer struct {
	parent *Buffer
	offset Rect
}

// NewSubBuffer creates a sub-buffer view into a region of this buffer.
func NewSubBuffer(b *Buffer, rect Rect) *SubBuffer {
	clipped := rect.Intersect(b.Bounds())
	return &SubBuffer{
		parent: b,
		offset: clipped,
	}
}

// Width returns the sub-buffer width.
func (s *SubBuffer) Width() int {
	return s.offset.Width
}

// Height returns the sub-buffer height.
func (s *SubBuffer) Height() int {
	return s.offset.Height
}

// Size returns the sub-buffer dimensions.
func (s *SubBuffer) Size() Size {
	return s.offset.Size
}

// Bounds returns the sub-buffer area (relative coordinates starting at 0,0).
func (s *SubBuffer) Bounds() Rect {
	return Rect{Position{X: 0, Y: 0}, s.offset.Size}
}

// GetCell returns the cell at the given position.
func (s *SubBuffer) GetCell(x, y int) Cell {
	if x < 0 || x >= s.offset.Width || y < 0 || y >= s.offset.Height {
		return EmptyCell
	}
	return s.parent.GetCell(s.offset.X+x, s.offset.Y+y)
}

// SetCells writes a slice of cells at position (x, y) relative to sub-buffer origin.
// Cells that extend beyond the sub-buffer width are clipped.
func (s *SubBuffer) SetCells(x, y int, cells []Cell) {
	if y < 0 || y >= s.offset.Height || len(cells) == 0 {
		return
	}

	// Handle negative x by skipping cells
	if x < 0 {
		skip := -x
		if skip >= len(cells) {
			return
		}
		cells = cells[skip:]
		x = 0
	}

	// Clip to sub-buffer width
	available := s.offset.Width - x
	if available <= 0 {
		return
	}
	if len(cells) > available {
		cells = cells[:available]
	}

	s.parent.SetCells(s.offset.X+x, s.offset.Y+y, cells)
}

// SetString writes a string at the given position.
func (s *SubBuffer) SetString(x, y int, str string, style lipgloss.Style) {
	if y < 0 || y >= s.offset.Height || len(str) == 0 {
		return
	}

	runes := []rune(str)

	// Build cells slice
	cells := make([]Cell, len(runes))
	for i, r := range runes {
		cells[i] = Cell{Rune: r, Style: style}
	}

	s.SetCells(x, y, cells)
}

// SetStringTruncated writes a string, truncating with ellipsis if needed.
func (s *SubBuffer) SetStringTruncated(x, y int, str string, maxWidth int, style lipgloss.Style) {
	if y < 0 || y >= s.offset.Height || maxWidth <= 0 {
		return
	}

	runes := []rune(str)
	if len(runes) > maxWidth {
		if maxWidth > 1 {
			runes = append(runes[:maxWidth-1], '…')
		} else {
			runes = runes[:maxWidth]
		}
	}

	s.SetString(x, y, string(runes), style)
}

// Fill fills a rectangular region.
func (s *SubBuffer) Fill(rect Rect, cell Cell) {
	clipped := rect.Intersect(s.Bounds())
	if clipped.Width == 0 || clipped.Height == 0 {
		return
	}

	// Build a row of cells to copy
	row := make([]Cell, clipped.Width)
	for i := range row {
		row[i] = cell
	}

	for y := clipped.Y; y < clipped.Y+clipped.Height; y++ {
		s.SetCells(clipped.X, y, row)
	}
}

// Clear fills the sub-buffer with empty cells.
func (s *SubBuffer) Clear() {
	s.Fill(s.Bounds(), EmptyCell)
}

// AbsoluteOffset returns the absolute offset of this sub-buffer within the root buffer.
// This is useful for widgets that need to track their screen-space position.
func (s *SubBuffer) AbsoluteOffset() (x, y int) {
	return s.offset.X, s.offset.Y
}

// Sub creates a nested sub-buffer view within this sub-buffer.
// The rect is relative to this sub-buffer's origin.
func (s *SubBuffer) Sub(rect Rect) *SubBuffer {
	// Convert relative coordinates to absolute coordinates in the parent buffer
	absRect := Rect{
		Position{
			X: s.offset.X + rect.X,
			Y: s.offset.Y + rect.Y,
		},
		Size{
			Width:  rect.Width,
			Height: rect.Height,
		},
	}
	// Clip to our bounds
	clipped := absRect.Intersect(Rect{
		Position{
			X: s.offset.X,
			Y: s.offset.Y,
		},
		Size{
			Width:  s.offset.Width,
			Height: s.offset.Height,
		},
	})
	return &SubBuffer{
		parent: s.parent,
		offset: clipped,
	}
}

// InvertRow applies Reverse styling to all cells in the specified row (relative to sub-buffer).
// This is used for selection highlighting as an overlay effect.
func (s *SubBuffer) InvertRow(row int) {
	if row < 0 || row >= s.offset.Height {
		return
	}
	cells := make([]Cell, s.offset.Width)
	for x := 0; x < s.offset.Width; x++ {
		cells[x] = s.GetCell(x, row)
		cells[x].Style = cells[x].Style.Reverse(true)
	}
	s.SetCells(0, row, cells)
}
