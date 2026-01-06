package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Text is a simple widget that displays styled text.
type Text struct {
	BaseWidget
	content string
	style   lipgloss.Style
	align   Alignment
}

// Alignment specifies text alignment.
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// NewText creates a new text widget.
func NewText(content string) *Text {
	t := &Text{
		BaseWidget: NewBaseWidget(),
		content:    content,
		style:      lipgloss.NewStyle(),
		align:      AlignLeft,
	}
	t.SetFocusable(false)
	return t
}

// SetContent sets the text content.
func (t *Text) SetContent(content string) {
	t.content = content
}

// Content returns the text content.
func (t *Text) Content() string {
	return t.content
}

// SetStyle sets the text style.
func (t *Text) SetStyle(style lipgloss.Style) {
	t.style = style
}

// Style returns the text style.
func (t *Text) Style() lipgloss.Style {
	return t.style
}

// SetAlignment sets the text alignment.
func (t *Text) SetAlignment(align Alignment) {
	t.align = align
}

// Constraints returns size constraints based on content.
func (t *Text) Constraints() Constraints {
	lines := strings.Split(t.content, "\n")
	maxWidth := 0
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	return DefaultConstraints().
		WithMinSize(1, 1).
		WithPreferredSize(maxWidth, len(lines))
}

// Render renders the text to the buffer.
func (t *Text) Render(buf *SubBuffer) {
	lines := strings.Split(t.content, "\n")

	for y, line := range lines {
		if y >= buf.Height() {
			break
		}

		// Calculate x position based on alignment
		x := 0
		lineLen := len([]rune(line))
		switch t.align {
		case AlignCenter:
			x = (buf.Width() - lineLen) / 2
		case AlignRight:
			x = buf.Width() - lineLen
		}

		if x < 0 {
			x = 0
		}

		buf.SetStringTruncated(x, y, line, buf.Width()-x, t.style)
	}
}

// StatusBar is a widget that displays status information.
type StatusBar struct {
	BaseWidget
	left   string
	center string
	right  string
	style  lipgloss.Style
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	sb := &StatusBar{
		BaseWidget: NewBaseWidget(),
		style: lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")),
	}
	sb.SetFocusable(false)
	sb.SetConstraints(DefaultConstraints().WithMinSize(1, 1).WithPreferredSize(0, 1))
	return sb
}

// SetLeft sets the left section text.
func (s *StatusBar) SetLeft(text string) {
	s.left = text
}

// SetCenter sets the center section text.
func (s *StatusBar) SetCenter(text string) {
	s.center = text
}

// SetRight sets the right section text.
func (s *StatusBar) SetRight(text string) {
	s.right = text
}

// SetStyle sets the status bar style.
func (s *StatusBar) SetStyle(style lipgloss.Style) {
	s.style = style
}

// Render renders the status bar.
func (s *StatusBar) Render(buf *SubBuffer) {
	width := buf.Width()

	// Fill background
	for x := 0; x < width; x++ {
		buf.SetCell(x, 0, Cell{Rune: ' ', Style: s.style})
	}

	// Render left
	if s.left != "" {
		buf.SetStringTruncated(0, 0, s.left, width/3, s.style)
	}

	// Render center
	if s.center != "" {
		centerX := (width - len(s.center)) / 2
		if centerX < 0 {
			centerX = 0
		}
		buf.SetStringTruncated(centerX, 0, s.center, width/3, s.style)
	}

	// Render right
	if s.right != "" {
		rightX := width - len(s.right)
		if rightX < 0 {
			rightX = 0
		}
		buf.SetStringTruncated(rightX, 0, s.right, width-rightX, s.style)
	}
}

// Spacer is a widget that takes up space but renders nothing.
// Useful for flexible spacing in layouts.
type Spacer struct {
	BaseWidget
}

// NewSpacer creates a new spacer with the given stretch factor.
func NewSpacer(stretch int) *Spacer {
	s := &Spacer{
		BaseWidget: NewBaseWidget(),
	}
	s.SetFocusable(false)
	s.SetConstraints(DefaultConstraints().WithStretch(stretch, stretch))
	return s
}

// NewHSpacer creates a horizontal spacer.
func NewHSpacer(stretch int) *Spacer {
	s := &Spacer{
		BaseWidget: NewBaseWidget(),
	}
	s.SetFocusable(false)
	s.SetConstraints(DefaultConstraints().WithStretch(stretch, 0))
	return s
}

// NewVSpacer creates a vertical spacer.
func NewVSpacer(stretch int) *Spacer {
	s := &Spacer{
		BaseWidget: NewBaseWidget(),
	}
	s.SetFocusable(false)
	s.SetConstraints(DefaultConstraints().WithStretch(0, stretch))
	return s
}

// Render is a no-op for spacer.
func (s *Spacer) Render(buf *SubBuffer) {
	// Intentionally empty - spacers are invisible
}
