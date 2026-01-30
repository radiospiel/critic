package teapot

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BorderStyle defines the style of a border edge.
type BorderStyle int

const (
	BorderNone   BorderStyle = iota // No border, takes no space
	BorderBlank                     // Invisible border, takes space
	BorderSingle                    // Single line: ─ │ ┌ ┐ └ ┘
	BorderDouble                    // Double line: ═ ║ ╔ ╗ ╚ ╝
	BorderRound                     // Rounded corners: ─ │ ╭ ╮ ╰ ╯
	BorderThick                     // Thick line: ━ ┃ ┏ ┓ ┗ ┛
)

// Border defines the border configuration for a widget.
// Order is: Top, Right, Bottom, Left (like CSS)
type Border struct {
	Top    BorderStyle
	Right  BorderStyle
	Bottom BorderStyle
	Left   BorderStyle
	Style  lipgloss.Style // Color/styling for the border
}

// NoBorder returns a border configuration with no borders.
func NoBorder() Border {
	return Border{
		Top:    BorderNone,
		Right:  BorderNone,
		Bottom: BorderNone,
		Left:   BorderNone,
		Style:  lipgloss.NewStyle(),
	}
}

// SingleBorder returns a border with single lines on all sides.
func SingleBorder() Border {
	return Border{
		Top:    BorderSingle,
		Right:  BorderSingle,
		Bottom: BorderSingle,
		Left:   BorderSingle,
		Style:  lipgloss.NewStyle(),
	}
}

// DoubleBorder returns a border with double lines on all sides.
func DoubleBorder() Border {
	return Border{
		Top:    BorderDouble,
		Right:  BorderDouble,
		Bottom: BorderDouble,
		Left:   BorderDouble,
		Style:  lipgloss.NewStyle(),
	}
}

// RoundBorder returns a border with rounded corners.
func RoundBorder() Border {
	return Border{
		Top:    BorderRound,
		Right:  BorderRound,
		Bottom: BorderRound,
		Left:   BorderRound,
		Style:  lipgloss.NewStyle(),
	}
}

// ThickBorder returns a border with thick lines on all sides.
func ThickBorder() Border {
	return Border{
		Top:    BorderThick,
		Right:  BorderThick,
		Bottom: BorderThick,
		Left:   BorderThick,
		Style:  lipgloss.NewStyle(),
	}
}

// WithStyle returns a copy of the border with the given style.
func (b Border) WithStyle(style lipgloss.Style) Border {
	b.Style = style
	return b
}

// WithTop returns a copy with the top border style changed.
func (b Border) WithTop(style BorderStyle) Border {
	b.Top = style
	return b
}

// WithRight returns a copy with the right border style changed.
func (b Border) WithRight(style BorderStyle) Border {
	b.Right = style
	return b
}

// WithBottom returns a copy with the bottom border style changed.
func (b Border) WithBottom(style BorderStyle) Border {
	b.Bottom = style
	return b
}

// WithLeft returns a copy with the left border style changed.
func (b Border) WithLeft(style BorderStyle) Border {
	b.Left = style
	return b
}

// HasBorder returns true if any side has a visible border.
func (b Border) HasBorder() bool {
	return b.Top != BorderNone || b.Right != BorderNone ||
		b.Bottom != BorderNone || b.Left != BorderNone
}

// TakesSpace returns true if the border takes up space (not BorderNone).
func (b Border) TakesSpace() bool {
	return b.HasBorder()
}

// TopWidth returns 1 if top border takes space, 0 otherwise.
func (b Border) TopWidth() int {
	if b.Top != BorderNone {
		return 1
	}
	return 0
}

// RightWidth returns 1 if right border takes space, 0 otherwise.
func (b Border) RightWidth() int {
	if b.Right != BorderNone {
		return 1
	}
	return 0
}

// BottomWidth returns 1 if bottom border takes space, 0 otherwise.
func (b Border) BottomWidth() int {
	if b.Bottom != BorderNone {
		return 1
	}
	return 0
}

// LeftWidth returns 1 if left border takes space, 0 otherwise.
func (b Border) LeftWidth() int {
	if b.Left != BorderNone {
		return 1
	}
	return 0
}

// borderChars holds the characters for each border style.
type borderChars struct {
	horizontal rune // ─
	vertical   rune // │
	topLeft    rune // ┌
	topRight   rune // ┐
	bottomLeft rune // └
	bottomRight rune // ┘
}

var borderCharSets = map[BorderStyle]borderChars{
	BorderBlank: {' ', ' ', ' ', ' ', ' ', ' '},
	BorderSingle: {'─', '│', '┌', '┐', '└', '┘'},
	BorderDouble: {'═', '║', '╔', '╗', '╚', '╝'},
	BorderRound:  {'─', '│', '╭', '╮', '╰', '╯'},
	BorderThick:  {'━', '┃', '┏', '┓', '┗', '┛'},
}

// getCorner returns the appropriate corner character based on the two adjacent border styles.
func getCorner(vertical, horizontal BorderStyle, position string) rune {
	// Use the "stronger" style for the corner
	style := vertical
	if horizontal > style {
		style = horizontal
	}
	if style == BorderNone {
		return ' '
	}
	chars := borderCharSets[style]
	switch position {
	case "topLeft":
		return chars.topLeft
	case "topRight":
		return chars.topRight
	case "bottomLeft":
		return chars.bottomLeft
	case "bottomRight":
		return chars.bottomRight
	}
	return ' '
}

// RenderBorder renders the border onto a buffer.
func RenderBorder(buf *SubBuffer, border Border) {
	if !border.HasBorder() {
		return
	}

	width := buf.Width()
	height := buf.Height()

	if width < 2 || height < 2 {
		return
	}

	// Get character sets
	topChars := borderCharSets[border.Top]
	bottomChars := borderCharSets[border.Bottom]
	leftChars := borderCharSets[border.Left]
	rightChars := borderCharSets[border.Right]

	// Draw corners (y values are valid since height >= 2, strings are single chars)
	if border.Top != BorderNone || border.Left != BorderNone {
		corner := getCorner(border.Left, border.Top, "topLeft")
		buf.setString(0, 0, string(corner), border.Style)
	}
	if border.Top != BorderNone || border.Right != BorderNone {
		corner := getCorner(border.Right, border.Top, "topRight")
		buf.setString(width-1, 0, string(corner), border.Style)
	}
	if border.Bottom != BorderNone || border.Left != BorderNone {
		corner := getCorner(border.Left, border.Bottom, "bottomLeft")
		buf.setString(0, height-1, string(corner), border.Style)
	}
	if border.Bottom != BorderNone || border.Right != BorderNone {
		corner := getCorner(border.Right, border.Bottom, "bottomRight")
		buf.setString(width-1, height-1, string(corner), border.Style)
	}

	// Draw top edge
	if border.Top != BorderNone {
		buf.SetString(1, 0, strings.Repeat(string(topChars.horizontal), width-2), border.Style)
	}

	// Draw bottom edge
	if border.Bottom != BorderNone {
		buf.SetString(1, height-1, strings.Repeat(string(bottomChars.horizontal), width-2), border.Style)
	}

	// Draw left edge (loop only executes when y values are valid)
	if border.Left != BorderNone {
		for y := 1; y < height-1; y++ {
			buf.setString(0, y, string(leftChars.vertical), border.Style)
		}
	}

	// Draw right edge (loop only executes when y values are valid)
	if border.Right != BorderNone {
		for y := 1; y < height-1; y++ {
			buf.setString(width-1, y, string(rightChars.vertical), border.Style)
		}
	}
}

// RenderBorderTitle renders a title centered on the top border.
func RenderBorderTitle(buf *SubBuffer, title string, style lipgloss.Style) {
	if title == "" {
		return
	}

	width := buf.Width()
	titleWithPadding := " " + title + " "
	titleRunes := []rune(titleWithPadding)
	titleX := (width - len(titleRunes)) / 2
	if titleX < 1 {
		titleX = 1
	}

	// Truncate if needed to fit within borders
	maxWidth := width - 2 - titleX
	if maxWidth <= 0 {
		return
	}
	if len(titleRunes) > maxWidth {
		titleRunes = titleRunes[:maxWidth]
	}

	buf.SetString(titleX, 0, string(titleRunes), style)
}

// RenderBorderFooter renders text centered on the bottom border.
func RenderBorderFooter(buf *SubBuffer, footer string, style lipgloss.Style) {
	if footer == "" {
		return
	}

	width := buf.Width()
	height := buf.Height()
	footerWithPadding := " " + footer + " "
	footerRunes := []rune(footerWithPadding)
	footerX := (width - len(footerRunes)) / 2
	if footerX < 1 {
		footerX = 1
	}

	// Truncate if needed to fit within borders
	maxWidth := width - 2 - footerX
	if maxWidth <= 0 {
		return
	}
	if len(footerRunes) > maxWidth {
		footerRunes = footerRunes[:maxWidth]
	}

	buf.SetString(footerX, height-1, string(footerRunes), style)
}
