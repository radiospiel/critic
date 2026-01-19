package teapot

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// PrintableWidth returns the visible width of a string, ignoring ANSI sequences.
func PrintableWidth(s string) int {
	return len([]rune(StripANSI(s)))
}

// ParseANSILine parses an ANSI-encoded line and returns cells with styles.
// This properly handles escape sequences and extracts visible characters with their styles.
func ParseANSILine(line string) []Cell {
	var cells []Cell
	var currentStyle lipgloss.Style

	i := 0
	runes := []rune(line)
	for i < len(runes) {
		r := runes[i]

		// Check for ANSI escape sequence
		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Find the end of the escape sequence
			j := i + 2
			for j < len(runes) && !isANSITerminator(runes[j]) {
				j++
			}
			if j < len(runes) {
				// Parse the escape sequence and update style
				seq := string(runes[i : j+1])
				currentStyle = applyANSISequence(currentStyle, seq)
				i = j + 1
				continue
			}
		}

		// Regular visible character
		cells = append(cells, Cell{Rune: r, Style: currentStyle})
		i++
	}

	return cells
}

// isANSITerminator returns true if r is an ANSI sequence terminator
func isANSITerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// applyANSISequence updates a style based on an ANSI escape sequence
func applyANSISequence(style lipgloss.Style, seq string) lipgloss.Style {
	// Extract parameters from sequence like "\x1b[38;5;220m"
	if len(seq) < 3 {
		return style
	}

	// Get the part between '[' and the terminator
	inner := seq[2 : len(seq)-1]
	terminator := seq[len(seq)-1]

	// Only handle SGR (Select Graphic Rendition) sequences ending in 'm'
	if terminator != 'm' {
		return style
	}

	// Reset
	if inner == "" || inner == "0" {
		return lipgloss.NewStyle()
	}

	// Parse semicolon-separated parameters
	params := strings.Split(inner, ";")
	i := 0
	for i < len(params) {
		p := params[i]
		switch p {
		case "0":
			style = lipgloss.NewStyle()
		case "1":
			style = style.Bold(true)
		case "2":
			style = style.Faint(true)
		case "3":
			style = style.Italic(true)
		case "4":
			style = style.Underline(true)
		case "5":
			style = style.Blink(true)
		case "7":
			style = style.Reverse(true)
		case "22":
			style = style.Bold(false).Faint(false)
		case "23":
			style = style.Italic(false)
		case "24":
			style = style.Underline(false)
		case "27":
			style = style.Reverse(false)
		case "38": // Foreground color
			if i+1 < len(params) {
				if params[i+1] == "5" && i+2 < len(params) {
					// 256 color: \x1b[38;5;COLORm
					style = style.Foreground(lipgloss.Color(params[i+2]))
					i += 2
				} else if params[i+1] == "2" && i+4 < len(params) {
					// RGB: \x1b[38;2;R;G;Bm
					style = style.Foreground(lipgloss.Color("#" + rgbToHex(params[i+2], params[i+3], params[i+4])))
					i += 4
				}
			}
		case "48": // Background color
			if i+1 < len(params) {
				if params[i+1] == "5" && i+2 < len(params) {
					style = style.Background(lipgloss.Color(params[i+2]))
					i += 2
				} else if params[i+1] == "2" && i+4 < len(params) {
					style = style.Background(lipgloss.Color("#" + rgbToHex(params[i+2], params[i+3], params[i+4])))
					i += 4
				}
			}
		case "39":
			// Default foreground - clear foreground
			style = style.UnsetForeground()
		case "49":
			// Default background - clear background
			style = style.UnsetBackground()
		default:
			// Basic colors 30-37 (foreground) and 40-47 (background)
			if len(p) > 0 {
				code := 0
				for _, c := range p {
					if c >= '0' && c <= '9' {
						code = code*10 + int(c-'0')
					}
				}
				if code >= 30 && code <= 37 {
					style = style.Foreground(lipgloss.Color(p))
				} else if code >= 40 && code <= 47 {
					style = style.Background(lipgloss.Color(p))
				} else if code >= 90 && code <= 97 {
					// Bright foreground
					style = style.Foreground(lipgloss.Color(p))
				} else if code >= 100 && code <= 107 {
					// Bright background
					style = style.Background(lipgloss.Color(p))
				}
			}
		}
		i++
	}

	return style
}

// rgbToHex converts r,g,b strings to a hex color string
func rgbToHex(r, g, b string) string {
	ri, _ := strconv.Atoi(r)
	gi, _ := strconv.Atoi(g)
	bi, _ := strconv.Atoi(b)
	return fmt.Sprintf("%02x%02x%02x", ri, gi, bi)
}

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
