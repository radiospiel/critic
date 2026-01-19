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

// noStyle is the default unstyled lipgloss style.
var noStyle = lipgloss.NewStyle()

// noStyleStr is the rendered representation of noStyle for fast comparison.
var noStyleStr = noStyle.Render("")

// EmptyCell is a cell with a space and no styling.
var EmptyCell = Cell{Rune: ' ', Style: noStyle}

func emptyRow(width int) []Cell {
	row := make([]Cell, width)
	for i := range row {
		row[i] = EmptyCell
	}
	return row
}

// precalculatedEmptyRow is a pre-allocated row of empty cells for fast copying.
var precalculatedEmptyRow = emptyRow(256)

// EmptyRow returns a slice of empty cells to copy from.
// For width <= 256, returns a slice of the preallocated empty row.
// For width > 256, allocates and fills a new row.
func EmptyRow(width int) []Cell {
	if width <= 256 {
		return precalculatedEmptyRow[:width]
	}

	return emptyRow(width)
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
	emptyRow := EmptyRow(width)

	for y := 0; y < height; y++ {
		cells[y] = make([]Cell, width)
		copy(cells[y], emptyRow)
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
	// TODO(bot) store bounds as rect
	return Rect{Position{X: 0, Y: 0}, Size{Width: b.width, Height: b.height}}
}

// Clear fills the entire buffer with empty cells.
func (b *Buffer) Clear() {
	if b.height == 0 || b.width == 0 {
		return
	}

	emptyRow := EmptyRow(b.width)
	for y := 0; y < b.height; y++ {
		copy(b.cells[y], emptyRow)
	}
}

// GetCell returns the cell at position (x, y).
// Out of bounds reads return an empty cell.
func (b *Buffer) GetCell(x, y int) Cell {
	if x < 0 || x >= b.width || y < 0 || y >= b.height {
		return EmptyCell
	}
	return b.cells[y][x]
}

// SetCells writes a slice of cells at position (x, y).
// Cells that extend beyond the buffer width are clipped.
// Uses copy for efficiency when possible.
func (b *Buffer) SetCells(x, y int, cells []Cell) {
	if y < 0 || y >= b.height || len(cells) == 0 {
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

	// Clip to buffer width
	available := b.width - x
	if available <= 0 {
		return
	}
	if len(cells) > available {
		cells = cells[:available]
	}

	copy(b.cells[y][x:], cells)
}

// SetString writes a string at position (x, y) with the given style.
// Characters that extend beyond the buffer width are clipped.
func (b *Buffer) SetString(x, y int, s string, style lipgloss.Style) {
	if y < 0 || y >= b.height || len(s) == 0 {
		return
	}

	runes := []rune(s)

	// Build cells slice
	cells := make([]Cell, len(runes))
	for i, r := range runes {
		cells[i] = Cell{Rune: r, Style: style}
	}

	b.SetCells(x, y, cells)
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
	if clipped.Width == 0 || clipped.Height == 0 {
		return
	}
	for y := clipped.Y; y < clipped.Y+clipped.Height; y++ {
		for x := clipped.X; x < clipped.X+clipped.Width; x++ {
			b.cells[y][x] = cell
		}
	}
}

// FillStyle fills a rectangular region with a style (keeping existing runes).
func (b *Buffer) FillStyle(rect Rect, style lipgloss.Style) {
	clipped := rect.Intersect(b.Bounds())
	if clipped.Width == 0 || clipped.Height == 0 {
		return
	}

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

	innerWidth := rect.Width - 2

	// Top edge: ┌───┐
	topEdge := "┌" + strings.Repeat("─", innerWidth) + "┐"
	b.SetString(rect.X, rect.Y, topEdge, style)

	// Bottom edge: └───┘
	bottomEdge := "└" + strings.Repeat("─", innerWidth) + "┘"
	b.SetString(rect.X, rect.Y+rect.Height-1, bottomEdge, style)

	// Left and right edges
	for y := rect.Y + 1; y < rect.Y+rect.Height-1; y++ {
		b.SetString(rect.X, y, "│", style)
		b.SetString(rect.X+rect.Width-1, y, "│", style)
	}
}

// DrawVerticalLine draws a vertical line at column x from y1 to y2.
func (b *Buffer) DrawVerticalLine(x, y1, y2 int, style lipgloss.Style) {
	for y := y1; y <= y2; y++ {
		b.SetString(x, y, "│", style)
	}
}

// DrawHorizontalLine draws a horizontal line at row y from x1 to x2.
func (b *Buffer) DrawHorizontalLine(y, x1, x2 int, style lipgloss.Style) {
	b.SetString(x1, y, strings.Repeat("─", x2-x1+1), style)
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

// InvertRow applies Reverse styling to all cells in the specified row.
// This is used for selection highlighting as an overlay effect.
func (b *Buffer) InvertRow(row int) {
	if row < 0 || row >= b.height {
		return
	}
	for x := 0; x < b.width; x++ {
		b.cells[row][x].Style = b.cells[row][x].Style.Reverse(true)
	}
}

// RenderToString renders the buffer to a string for terminal output.
// This is the final step before sending to the terminal.
func (b *Buffer) RenderToString() string {
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
		renderRow(b.cells[y], &sb)
	}

	return sb.String()
}

// renderRow renders a row of cells, grouping consecutive cells with the same style.
func renderRow(row []Cell, sb *strings.Builder) {
	if len(row) == 0 {
		return
	}

	var runes strings.Builder
	runes.Grow(len(row)) // Preallocate for worst case (all same style)
	currentStyleStr := row[0].Style.Render("")
	currentStyle := row[0].Style

	for _, cell := range row {
		cellStyleStr := cell.Style.Render("")
		if cellStyleStr != currentStyleStr {
			// Style changed - render accumulated runes and start new group
			writeStyled(sb, currentStyleStr, currentStyle, runes.String())
			runes.Reset()
			currentStyle = cell.Style
			currentStyleStr = cellStyleStr
		}
		runes.WriteRune(cell.Rune)
	}

	// Render final group
	if runes.Len() > 0 {
		writeStyled(sb, currentStyleStr, currentStyle, runes.String())
	}
}

// writeStyled writes text to sb, applying style only if it's not noStyle.
func writeStyled(sb *strings.Builder, styleStr string, style lipgloss.Style, text string) {
	if styleStr == noStyleStr {
		sb.WriteString(text)
	} else {
		sb.WriteString(style.Render(text))
	}
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
