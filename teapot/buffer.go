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
// This type is used as the public API for widgets to pass styled characters.
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

// StyleID is a compact reference to a style in the buffer's style table.
// Using uint16 allows up to 65536 unique styles, which is more than sufficient
// for any TUI application while reducing per-cell memory from 552 bytes to 2 bytes.
type StyleID uint16

// noStyleID is the StyleID for the default unstyled style (always ID 0).
const noStyleID StyleID = 0

// styleKey computes a string key for a style based on its visual properties.
// Uses fmt.Sprintf with %p to get pointer identity for colors (no RGBA() allocation).
// This is much faster than style.Render("").
func styleKey(style lipgloss.Style) string {
	// Pack attributes into a single byte (7 bits for 7 attributes)
	var attrs byte
	if style.GetBold() {
		attrs |= 1 << 0
	}
	if style.GetItalic() {
		attrs |= 1 << 1
	}
	if style.GetUnderline() {
		attrs |= 1 << 2
	}
	if style.GetReverse() {
		attrs |= 1 << 3
	}
	if style.GetFaint() {
		attrs |= 1 << 4
	}
	if style.GetBlink() {
		attrs |= 1 << 5
	}
	if style.GetStrikethrough() {
		attrs |= 1 << 6
	}

	// Use %v for colors - this uses their String() method which doesn't allocate
	// like RGBA() does. For lipgloss.Color this returns the color value string.
	fg := style.GetForeground()
	bg := style.GetBackground()

	return fmt.Sprintf("%d|%v|%v", attrs, fg, bg)
}

// styleTable manages deduplicated styles for a buffer.
// Styles are looked up by their properties to ensure
// visually identical styles share the same ID.
type styleTable struct {
	styles []lipgloss.Style
	lookup map[string]StyleID
}

// noStyleKey is the pre-computed key for noStyle for fast comparison.
var noStyleKey = styleKey(noStyle)

// newStyleTable creates a new style table with noStyle pre-registered at ID 0.
func newStyleTable() *styleTable {
	st := &styleTable{
		styles: make([]lipgloss.Style, 1, 32), // Pre-allocate for typical usage
		lookup: make(map[string]StyleID, 32),
	}
	// Register noStyle at ID 0
	st.styles[0] = noStyle
	st.lookup[noStyleKey] = noStyleID
	return st
}

// intern returns the StyleID for a style, adding it to the table if new.
func (st *styleTable) intern(style lipgloss.Style) StyleID {
	key := styleKey(style)
	// Fast path for noStyle (very common)
	if key == noStyleKey {
		return noStyleID
	}
	if id, ok := st.lookup[key]; ok {
		return id
	}
	id := StyleID(len(st.styles))
	st.styles = append(st.styles, style)
	st.lookup[key] = id
	return id
}

// get returns the style for a given StyleID.
func (st *styleTable) get(id StyleID) lipgloss.Style {
	if int(id) >= len(st.styles) {
		return noStyle
	}
	return st.styles[id]
}

// getRendered returns the rendered string for a given StyleID (for fast comparison).
func (st *styleTable) getRendered(id StyleID) string {
	if int(id) >= len(st.styles) {
		return noStyleStr
	}
	return st.styles[id].Render("")
}

// clone creates a deep copy of the style table.
func (st *styleTable) clone() *styleTable {
	newST := &styleTable{
		styles: make([]lipgloss.Style, len(st.styles)),
		lookup: make(map[string]StyleID, len(st.lookup)),
	}
	copy(newST.styles, st.styles)
	for k, v := range st.lookup {
		newST.lookup[k] = v
	}
	return newST
}

// Line represents a single row in the buffer.
// It stores runes and style IDs separately for memory efficiency.
type Line struct {
	Runes    []rune
	StyleIDs []StyleID
}

// newLine creates a new line with the given width, filled with spaces and noStyleID.
func newLine(width int) Line {
	line := Line{
		Runes:    make([]rune, width),
		StyleIDs: make([]StyleID, width),
	}
	for i := 0; i < width; i++ {
		line.Runes[i] = ' '
		// StyleIDs[i] is already 0 (noStyleID) from make()
	}
	return line
}

// clear resets the line to spaces with noStyleID.
func (l *Line) clear() {
	for i := range l.Runes {
		l.Runes[i] = ' '
		l.StyleIDs[i] = noStyleID
	}
}

// clone creates a deep copy of the line.
func (l *Line) clone() Line {
	newLine := Line{
		Runes:    make([]rune, len(l.Runes)),
		StyleIDs: make([]StyleID, len(l.StyleIDs)),
	}
	copy(newLine.Runes, l.Runes)
	copy(newLine.StyleIDs, l.StyleIDs)
	return newLine
}

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
//
// Internally, the buffer stores Lines (runes + style IDs) for efficiency:
// - runes: 4 bytes per cell (character content)
// - styleIDs: 2 bytes per cell (compact reference to style table)
// This reduces memory usage from ~560 bytes/cell to ~6 bytes/cell.
type Buffer struct {
	lines  []Line
	styles *styleTable
	width  int
	height int
}

// NewBuffer creates a new buffer with the given dimensions.
func NewBuffer(width, height int) *Buffer {
	if width <= 0 || height <= 0 {
		return &Buffer{width: 0, height: 0, styles: newStyleTable()}
	}

	lines := make([]Line, height)
	for y := 0; y < height; y++ {
		lines[y] = newLine(width)
	}

	return &Buffer{
		lines:  lines,
		styles: newStyleTable(),
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
	if b.height == 0 || b.width == 0 {
		return
	}

	for y := 0; y < b.height; y++ {
		b.lines[y].clear()
	}
}

// GetCell returns the cell at position (x, y).
// Out of bounds reads return an empty cell.
func (b *Buffer) GetCell(x, y int) Cell {
	if x < 0 || x >= b.width || y < 0 || y >= b.height {
		return EmptyCell
	}
	return Cell{
		Rune:  b.lines[y].Runes[x],
		Style: b.styles.get(b.lines[y].StyleIDs[x]),
	}
}

// SetCells writes a slice of cells at position (x, y).
// Cells that extend beyond the buffer width are clipped.
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

	// Write runes and style IDs separately
	line := &b.lines[y]
	for i, cell := range cells {
		line.Runes[x+i] = cell.Rune
		line.StyleIDs[x+i] = b.styles.intern(cell.Style)
	}
}

// SetString writes a string at position (x, y) with the given style.
// Characters that extend beyond the buffer width are clipped.
func (b *Buffer) SetString(x, y int, s string, style lipgloss.Style) {
	if y < 0 || y >= b.height || len(s) == 0 {
		return
	}

	// Handle negative x by skipping runes
	runes := []rune(s)
	if x < 0 {
		skip := -x
		if skip >= len(runes) {
			return
		}
		runes = runes[skip:]
		x = 0
	}

	// Clip to buffer width
	available := b.width - x
	if available <= 0 {
		return
	}
	if len(runes) > available {
		runes = runes[:available]
	}

	// Intern the style once for all characters
	styleID := b.styles.intern(style)

	// Write runes and styleIDs directly
	line := &b.lines[y]
	for i, r := range runes {
		line.Runes[x+i] = r
		line.StyleIDs[x+i] = styleID
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
	if clipped.Width == 0 || clipped.Height == 0 {
		return
	}
	styleID := b.styles.intern(cell.Style)
	for y := clipped.Y; y < clipped.Y+clipped.Height; y++ {
		line := &b.lines[y]
		for x := clipped.X; x < clipped.X+clipped.Width; x++ {
			line.Runes[x] = cell.Rune
			line.StyleIDs[x] = styleID
		}
	}
}

// FillStyle fills a rectangular region with a style (keeping existing runes).
func (b *Buffer) FillStyle(rect Rect, style lipgloss.Style) {
	clipped := rect.Intersect(b.Bounds())
	if clipped.Width == 0 || clipped.Height == 0 {
		return
	}

	// Intern the style once for the entire region
	styleID := b.styles.intern(style)
	for y := clipped.Y; y < clipped.Y+clipped.Height; y++ {
		line := &b.lines[y]
		for x := clipped.X; x < clipped.X+clipped.Width; x++ {
			line.StyleIDs[x] = styleID
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
		srcLine := &src.lines[y]
		dstLine := &b.lines[dy]
		for x := 0; x < src.width; x++ {
			dx := destX + x
			if dx < 0 || dx >= b.width {
				continue
			}
			dstLine.Runes[dx] = srcLine.Runes[x]
			// Re-intern the style from src into our style table
			srcStyle := src.styles.get(srcLine.StyleIDs[x])
			dstLine.StyleIDs[dx] = b.styles.intern(srcStyle)
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
		srcLine := &src.lines[sy]
		dstLine := &b.lines[dy]
		for x := 0; x < srcRect.Width; x++ {
			sx := srcRect.X + x
			dx := destX + x
			if sx < 0 || sx >= src.width || dx < 0 || dx >= b.width {
				continue
			}
			dstLine.Runes[dx] = srcLine.Runes[sx]
			// Re-intern the style from src into our style table
			srcStyle := src.styles.get(srcLine.StyleIDs[sx])
			dstLine.StyleIDs[dx] = b.styles.intern(srcStyle)
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

	// Handle negative x by skipping runes
	runes := []rune(str)
	if x < 0 {
		skip := -x
		if skip >= len(runes) {
			return
		}
		runes = runes[skip:]
		x = 0
	}

	// Clip to sub-buffer width
	available := s.offset.Width - x
	if available <= 0 {
		return
	}
	if len(runes) > available {
		runes = runes[:available]
	}

	// Write directly to parent buffer (more efficient than creating Cell slice)
	absX := s.offset.X + x
	absY := s.offset.Y + y
	styleID := s.parent.styles.intern(style)
	line := &s.parent.lines[absY]
	for i, r := range runes {
		line.Runes[absX+i] = r
		line.StyleIDs[absX+i] = styleID
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
		X:      s.offset.X + rect.X,
		Y:      s.offset.Y + rect.Y,
		Width:  rect.Width,
		Height: rect.Height,
	}
	// Clip to our bounds
	clipped := absRect.Intersect(Rect{
		X:      s.offset.X,
		Y:      s.offset.Y,
		Width:  s.offset.Width,
		Height: s.offset.Height,
	})
	return &SubBuffer{
		parent: s.parent,
		offset: clipped,
	}
}

// InvertRow applies Reverse styling to all cells in the specified row.
// This is used for selection highlighting as an overlay effect.
func (b *Buffer) InvertRow(row int) {
	if row < 0 || row >= b.height {
		return
	}
	line := &b.lines[row]
	for x := 0; x < b.width; x++ {
		// Get the current style, apply reverse, and intern the new style
		currentStyle := b.styles.get(line.StyleIDs[x])
		invertedStyle := currentStyle.Reverse(true)
		line.StyleIDs[x] = b.styles.intern(invertedStyle)
	}
}

// InvertRow applies Reverse styling to all cells in the specified row (relative to sub-buffer).
// This is used for selection highlighting as an overlay effect.
func (s *SubBuffer) InvertRow(row int) {
	if row < 0 || row >= s.offset.Height {
		return
	}
	// Delegate to parent's row inversion for the absolute row
	absY := s.offset.Y + row
	if absY < 0 || absY >= s.parent.height {
		return
	}
	line := &s.parent.lines[absY]
	// Only invert cells within our horizontal bounds
	for x := 0; x < s.offset.Width; x++ {
		absX := s.offset.X + x
		if absX < 0 || absX >= s.parent.width {
			continue
		}
		currentStyle := s.parent.styles.get(line.StyleIDs[absX])
		invertedStyle := currentStyle.Reverse(true)
		line.StyleIDs[absX] = s.parent.styles.intern(invertedStyle)
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
		b.renderRow(y, &sb)
	}

	return sb.String()
}

// String returns the buffer content as a string (alias for RenderToString).
func (b *Buffer) String() string {
	return b.RenderToString()
}

// renderRow renders a row, grouping consecutive cells with the same style.
// This is now a method on Buffer to access the style table.
func (b *Buffer) renderRow(y int, sb *strings.Builder) {
	if b.width == 0 {
		return
	}

	line := &b.lines[y]
	var runesBuf strings.Builder
	runesBuf.Grow(b.width) // Preallocate for worst case (all same style)

	currentStyleID := line.StyleIDs[0]

	for x := 0; x < b.width; x++ {
		styleID := line.StyleIDs[x]
		if styleID != currentStyleID {
			// Style changed - render accumulated runes and start new group
			b.writeStyledByID(sb, currentStyleID, runesBuf.String())
			runesBuf.Reset()
			currentStyleID = styleID
		}
		runesBuf.WriteRune(line.Runes[x])
	}

	// Render final group
	if runesBuf.Len() > 0 {
		b.writeStyledByID(sb, currentStyleID, runesBuf.String())
	}
}

// writeStyledByID writes text to sb, applying style only if it's not noStyle.
func (b *Buffer) writeStyledByID(sb *strings.Builder, styleID StyleID, text string) {
	if styleID == noStyleID {
		sb.WriteString(text)
	} else {
		style := b.styles.get(styleID)
		sb.WriteString(style.Render(text))
	}
}

// Equals compares two buffers for equality.
// Compares both runes and style IDs for complete equality.
func (b *Buffer) Equals(other *Buffer) bool {
	if b.width != other.width || b.height != other.height {
		return false
	}
	for y := 0; y < b.height; y++ {
		bLine := &b.lines[y]
		oLine := &other.lines[y]
		for x := 0; x < b.width; x++ {
			if bLine.Runes[x] != oLine.Runes[x] {
				return false
			}
			// Compare styles by their rendered representation
			// (different style tables may have different IDs for the same style)
			if b.styles.getRendered(bLine.StyleIDs[x]) != other.styles.getRendered(oLine.StyleIDs[x]) {
				return false
			}
		}
	}
	return true
}

// EqualsRunes compares only the rune content of two buffers.
// This is faster than Equals when style comparison is not needed.
func (b *Buffer) EqualsRunes(other *Buffer) bool {
	if b.width != other.width || b.height != other.height {
		return false
	}
	for y := 0; y < b.height; y++ {
		bLine := &b.lines[y]
		oLine := &other.lines[y]
		for x := 0; x < b.width; x++ {
			if bLine.Runes[x] != oLine.Runes[x] {
				return false
			}
		}
	}
	return true
}

// Clone creates a deep copy of the buffer.
func (b *Buffer) Clone() *Buffer {
	if b.width <= 0 || b.height <= 0 {
		return &Buffer{width: 0, height: 0, styles: newStyleTable()}
	}

	clone := &Buffer{
		lines:  make([]Line, b.height),
		styles: b.styles.clone(),
		width:  b.width,
		height: b.height,
	}

	for y := 0; y < b.height; y++ {
		clone.lines[y] = b.lines[y].clone()
	}

	return clone
}
