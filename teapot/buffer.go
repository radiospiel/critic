package teapot

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Cell represents a single character cell in the terminal buffer.
// Each cell has a rune and an associated style.
type Cell struct {
	Rune  rune
	Style lipgloss.Style
}

// EmptyCell returns a cell with a space and no styling.
func EmptyCell() Cell {
	return Cell{Rune: ' ', Style: lipgloss.NewStyle()}
}

// Buffer is a 2D grid of cells representing the terminal display.
// Widgets render into buffers, and the compositor combines them.
type Buffer struct {
	cells  [][]Cell
	width  int
	height int
}

// NewBuffer creates a new buffer with the given dimensions.
func NewBuffer(width, height int) *Buffer {
	if width <= 0 || height <= 0 {
		return &Buffer{width: 0, height: 0}
	}

	cells := make([][]Cell, height)
	for y := 0; y < height; y++ {
		cells[y] = make([]Cell, width)
		for x := 0; x < width; x++ {
			cells[y][x] = EmptyCell()
		}
	}

	return &Buffer{
		cells:  cells,
		width:  width,
		height: height,
	}
}

// Width returns the buffer width.
func (b *Buffer) Width() int {
	return b.width
}

// Height returns the buffer height.
func (b *Buffer) Height() int {
	return b.height
}

// Size returns the buffer dimensions.
func (b *Buffer) Size() Size {
	return Size{Width: b.width, Height: b.height}
}

// Bounds returns the full buffer area as a Rect.
func (b *Buffer) Bounds() Rect {
	return Rect{X: 0, Y: 0, Width: b.width, Height: b.height}
}

// Clear fills the entire buffer with empty cells.
func (b *Buffer) Clear() {
	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			b.cells[y][x] = EmptyCell()
		}
	}
}

// SetCell sets the cell at position (x, y).
// Out of bounds writes are silently ignored.
func (b *Buffer) SetCell(x, y int, cell Cell) {
	if x < 0 || x >= b.width || y < 0 || y >= b.height {
		return
	}
	b.cells[y][x] = cell
}

// GetCell returns the cell at position (x, y).
// Out of bounds reads return an empty cell.
func (b *Buffer) GetCell(x, y int) Cell {
	if x < 0 || x >= b.width || y < 0 || y >= b.height {
		return EmptyCell()
	}
	return b.cells[y][x]
}

// SetString writes a string at position (x, y) with the given style.
// Characters that extend beyond the buffer width are clipped.
func (b *Buffer) SetString(x, y int, s string, style lipgloss.Style) {
	if y < 0 || y >= b.height {
		return
	}

	for _, r := range s {
		if x >= b.width {
			break
		}
		if x >= 0 {
			b.cells[y][x] = Cell{Rune: r, Style: style}
		}
		x++
	}
}

// SetStringTruncated writes a string, truncating with ellipsis if needed.
func (b *Buffer) SetStringTruncated(x, y int, s string, maxWidth int, style lipgloss.Style) {
	if y < 0 || y >= b.height || maxWidth <= 0 {
		return
	}

	runes := []rune(s)
	if len(runes) > maxWidth {
		if maxWidth > 1 {
			runes = append(runes[:maxWidth-1], '…')
		} else {
			runes = runes[:maxWidth]
		}
	}

	b.SetString(x, y, string(runes), style)
}

// Fill fills a rectangular region with the given cell.
func (b *Buffer) Fill(rect Rect, cell Cell) {
	clipped := rect.Intersect(b.Bounds())
	for y := clipped.Y; y < clipped.Y+clipped.Height; y++ {
		for x := clipped.X; x < clipped.X+clipped.Width; x++ {
			b.cells[y][x] = cell
		}
	}
}

// FillStyle fills a rectangular region with a style (keeping existing runes).
func (b *Buffer) FillStyle(rect Rect, style lipgloss.Style) {
	clipped := rect.Intersect(b.Bounds())
	for y := clipped.Y; y < clipped.Y+clipped.Height; y++ {
		for x := clipped.X; x < clipped.X+clipped.Width; x++ {
			b.cells[y][x].Style = style
		}
	}
}

// DrawBox draws a box border around the given rectangle.
func (b *Buffer) DrawBox(rect Rect, style lipgloss.Style) {
	if rect.Width < 2 || rect.Height < 2 {
		return
	}

	// Corners
	b.SetCell(rect.X, rect.Y, Cell{Rune: '┌', Style: style})
	b.SetCell(rect.X+rect.Width-1, rect.Y, Cell{Rune: '┐', Style: style})
	b.SetCell(rect.X, rect.Y+rect.Height-1, Cell{Rune: '└', Style: style})
	b.SetCell(rect.X+rect.Width-1, rect.Y+rect.Height-1, Cell{Rune: '┘', Style: style})

	// Top and bottom edges
	for x := rect.X + 1; x < rect.X+rect.Width-1; x++ {
		b.SetCell(x, rect.Y, Cell{Rune: '─', Style: style})
		b.SetCell(x, rect.Y+rect.Height-1, Cell{Rune: '─', Style: style})
	}

	// Left and right edges
	for y := rect.Y + 1; y < rect.Y+rect.Height-1; y++ {
		b.SetCell(rect.X, y, Cell{Rune: '│', Style: style})
		b.SetCell(rect.X+rect.Width-1, y, Cell{Rune: '│', Style: style})
	}
}

// DrawVerticalLine draws a vertical line at column x from y1 to y2.
func (b *Buffer) DrawVerticalLine(x, y1, y2 int, style lipgloss.Style) {
	for y := y1; y <= y2; y++ {
		b.SetCell(x, y, Cell{Rune: '│', Style: style})
	}
}

// DrawHorizontalLine draws a horizontal line at row y from x1 to x2.
func (b *Buffer) DrawHorizontalLine(y, x1, x2 int, style lipgloss.Style) {
	for x := x1; x <= x2; x++ {
		b.SetCell(x, y, Cell{Rune: '─', Style: style})
	}
}

// Blit copies the contents of another buffer into this one at the given offset.
// This is used by the compositor to combine widget buffers.
func (b *Buffer) Blit(src *Buffer, destX, destY int) {
	for y := 0; y < src.height; y++ {
		dy := destY + y
		if dy < 0 || dy >= b.height {
			continue
		}
		for x := 0; x < src.width; x++ {
			dx := destX + x
			if dx < 0 || dx >= b.width {
				continue
			}
			b.cells[dy][dx] = src.cells[y][x]
		}
	}
}

// BlitRect copies a rectangular region from another buffer.
func (b *Buffer) BlitRect(src *Buffer, srcRect Rect, destX, destY int) {
	for y := 0; y < srcRect.Height; y++ {
		sy := srcRect.Y + y
		dy := destY + y
		if sy < 0 || sy >= src.height || dy < 0 || dy >= b.height {
			continue
		}
		for x := 0; x < srcRect.Width; x++ {
			sx := srcRect.X + x
			dx := destX + x
			if sx < 0 || sx >= src.width || dx < 0 || dx >= b.width {
				continue
			}
			b.cells[dy][dx] = src.cells[sy][sx]
		}
	}
}

// SubBuffer returns a view into a rectangular region of this buffer.
// Writes to the sub-buffer affect the original.
type SubBuffer struct {
	parent *Buffer
	offset Rect
}

// Sub creates a sub-buffer view into a region of this buffer.
func (b *Buffer) Sub(rect Rect) *SubBuffer {
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
	return Size{Width: s.offset.Width, Height: s.offset.Height}
}

// Bounds returns the sub-buffer area (relative coordinates starting at 0,0).
func (s *SubBuffer) Bounds() Rect {
	return Rect{X: 0, Y: 0, Width: s.offset.Width, Height: s.offset.Height}
}

// SetCell sets a cell at the given position (relative to sub-buffer origin).
func (s *SubBuffer) SetCell(x, y int, cell Cell) {
	if x < 0 || x >= s.offset.Width || y < 0 || y >= s.offset.Height {
		return
	}
	s.parent.SetCell(s.offset.X+x, s.offset.Y+y, cell)
}

// GetCell returns the cell at the given position.
func (s *SubBuffer) GetCell(x, y int) Cell {
	if x < 0 || x >= s.offset.Width || y < 0 || y >= s.offset.Height {
		return EmptyCell()
	}
	return s.parent.GetCell(s.offset.X+x, s.offset.Y+y)
}

// SetString writes a string at the given position.
func (s *SubBuffer) SetString(x, y int, str string, style lipgloss.Style) {
	if y < 0 || y >= s.offset.Height {
		return
	}
	for _, r := range str {
		if x >= s.offset.Width {
			break
		}
		if x >= 0 {
			s.SetCell(x, y, Cell{Rune: r, Style: style})
		}
		x++
	}
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
	for y := clipped.Y; y < clipped.Y+clipped.Height; y++ {
		for x := clipped.X; x < clipped.X+clipped.Width; x++ {
			s.SetCell(x, y, cell)
		}
	}
}

// Clear fills the sub-buffer with empty cells.
func (s *SubBuffer) Clear() {
	s.Fill(s.Bounds(), EmptyCell())
}

// String renders the buffer to a string for terminal output.
// This is the final step before sending to the terminal.
func (b *Buffer) String() string {
	if b.height == 0 || b.width == 0 {
		return ""
	}

	var sb strings.Builder
	// Pre-allocate approximate size
	sb.Grow(b.width * b.height * 4)

	for y := 0; y < b.height; y++ {
		if y > 0 {
			sb.WriteString("\n")
		}
		for x := 0; x < b.width; x++ {
			cell := b.cells[y][x]
			styled := cell.Style.Render(string(cell.Rune))
			sb.WriteString(styled)
		}
	}

	return sb.String()
}

// Equals compares two buffers for equality.
// Used for differential rendering. Note: only compares runes, not styles,
// since lipgloss.Style contains non-comparable fields.
func (b *Buffer) Equals(other *Buffer) bool {
	if b.width != other.width || b.height != other.height {
		return false
	}
	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			if b.cells[y][x].Rune != other.cells[y][x].Rune {
				return false
			}
		}
	}
	return true
}

// Clone creates a deep copy of the buffer.
func (b *Buffer) Clone() *Buffer {
	clone := NewBuffer(b.width, b.height)
	for y := 0; y < b.height; y++ {
		copy(clone.cells[y], b.cells[y])
	}
	return clone
}
