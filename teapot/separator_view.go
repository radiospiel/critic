package teapot

import "github.com/charmbracelet/lipgloss"

// SeparatorView renders a horizontal line separator.
type SeparatorView struct {
	BaseView
	char  rune
	style lipgloss.Style
}

// NewSeparatorView creates a new horizontal separator view.
func NewSeparatorView() *SeparatorView {
	return &SeparatorView{
		BaseView: NewBaseView(),
		char:     '─',
		style:    lipgloss.NewStyle().Faint(true),
	}
}

// SetChar sets the character used for the separator line.
func (s *SeparatorView) SetChar(r rune) {
	s.char = r
}

// SetStyle sets the style for the separator line.
func (s *SeparatorView) SetStyle(style lipgloss.Style) {
	s.style = style
}

// Constraints returns the separator's constraints (1 row height).
func (s *SeparatorView) Constraints() Constraints {
	return Constraints{
		MinHeight:       1,
		MaxHeight:       1,
		PreferredHeight: 1,
	}
}

// Render renders the separator line.
func (s *SeparatorView) Render(buf *SubBuffer) {
	width := buf.Width()
	rowCells := make([]Cell, width)
	for x := range width {
		rowCells[x] = Cell{Rune: s.char, Style: s.style}
	}
	buf.SetCells(0, 0, rowCells)
}
