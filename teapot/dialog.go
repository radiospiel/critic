package teapot

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DialogResult represents the result of a dialog interaction
type DialogResult int

const (
	DialogNone DialogResult = iota
	DialogOK
	DialogCancel
)

// Dialog is a modal dialog with OK and Cancel buttons.
// OK is triggered by Enter/Return, Cancel by Escape.
type Dialog struct {
	BaseWidget
	content     Widget
	title       string
	okLabel     string
	cancelLabel string
	result      DialogResult

	// Callbacks
	onOK     func()
	onCancel func()

	// Styles
	titleStyle      lipgloss.Style
	buttonStyle     lipgloss.Style
	buttonOKStyle   lipgloss.Style
	buttonHintStyle lipgloss.Style
	borderStyle     lipgloss.Style
}

// NewDialog creates a new dialog with the given content and title.
func NewDialog(content Widget, title string) *Dialog {
	d := &Dialog{
		BaseWidget:  NewBaseWidget(),
		content:     content,
		title:       title,
		okLabel:     "OK",
		cancelLabel: "Cancel",
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")),
		buttonStyle: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#666"}),
		buttonOKStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")),
		buttonHintStyle: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666", Dark: "#888"}),
		borderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")),
	}
	if content != nil {
		content.SetParent(d)
	}
	return d
}

// SetTitle sets the dialog title.
func (d *Dialog) SetTitle(title string) {
	d.title = title
}

// SetLabels sets the OK and Cancel button labels.
func (d *Dialog) SetLabels(ok, cancel string) {
	d.okLabel = ok
	d.cancelLabel = cancel
}

// OnOK sets the callback for when OK is pressed.
func (d *Dialog) OnOK(fn func()) {
	d.onOK = fn
}

// OnCancel sets the callback for when Cancel is pressed.
func (d *Dialog) OnCancel(fn func()) {
	d.onCancel = fn
}

// Result returns the last dialog result.
func (d *Dialog) Result() DialogResult {
	return d.result
}

// ResetResult resets the dialog result to None.
func (d *Dialog) ResetResult() {
	d.result = DialogNone
}

// Content returns the dialog's content widget.
func (d *Dialog) Content() Widget {
	return d.content
}

// SetContent sets the dialog's content widget.
func (d *Dialog) SetContent(w Widget) {
	if d.content != nil {
		d.content.SetParent(nil)
	}
	d.content = w
	if w != nil {
		w.SetParent(d)
	}
}

// Children returns the content widget.
func (d *Dialog) Children() []Widget {
	if d.content != nil {
		return []Widget{d.content}
	}
	return nil
}

// SetBounds sets the dialog bounds and layouts the content.
func (d *Dialog) SetBounds(bounds Rect) {
	d.BaseWidget.SetBounds(bounds)

	if d.content == nil {
		return
	}

	// Content gets the inner area (inside border, above button row)
	// Border: 1 on each side, Button row: 1 line
	contentBounds := Rect{
		X:      bounds.X + 1,
		Y:      bounds.Y + 1,
		Width:  bounds.Width - 2,
		Height: bounds.Height - 3, // Top border, content, button row, bottom border
	}
	d.content.SetBounds(contentBounds)
}

// Render renders the dialog with a bordered box and centered title.
func (d *Dialog) Render(buf *SubBuffer) {
	width := buf.Width()
	height := buf.Height()

	if width < 4 || height < 3 {
		return
	}

	// Draw border
	// Top-left corner
	buf.SetCell(0, 0, Cell{Rune: '┌', Style: d.borderStyle})
	// Top-right corner
	buf.SetCell(width-1, 0, Cell{Rune: '┐', Style: d.borderStyle})
	// Bottom-left corner
	buf.SetCell(0, height-1, Cell{Rune: '└', Style: d.borderStyle})
	// Bottom-right corner
	buf.SetCell(width-1, height-1, Cell{Rune: '┘', Style: d.borderStyle})

	// Top and bottom edges
	for x := 1; x < width-1; x++ {
		buf.SetCell(x, 0, Cell{Rune: '─', Style: d.borderStyle})
		buf.SetCell(x, height-1, Cell{Rune: '─', Style: d.borderStyle})
	}

	// Left and right edges
	for y := 1; y < height-1; y++ {
		buf.SetCell(0, y, Cell{Rune: '│', Style: d.borderStyle})
		buf.SetCell(width-1, y, Cell{Rune: '│', Style: d.borderStyle})
	}

	// Render title centered on top border
	if d.title != "" {
		title := " " + d.title + " "
		titleRunes := []rune(title)
		titleX := (width - len(titleRunes)) / 2
		if titleX < 1 {
			titleX = 1
		}
		for i, r := range titleRunes {
			x := titleX + i
			if x >= width-1 {
				break
			}
			buf.SetCell(x, 0, Cell{Rune: r, Style: d.titleStyle})
		}
	}

	// Render content
	if d.content != nil {
		contentSub := buf.parent.Sub(Rect{
			X:      buf.offset.X + 1,
			Y:      buf.offset.Y + 1,
			Width:  width - 2,
			Height: height - 3,
		})
		d.content.Render(contentSub)
	}

	// Render button hints on the bottom border
	hint := " Enter: " + d.okLabel + " │ Esc: " + d.cancelLabel + " "
	hintRunes := []rune(hint)
	hintX := (width - len(hintRunes)) / 2
	if hintX < 1 {
		hintX = 1
	}
	for i, r := range hintRunes {
		x := hintX + i
		if x >= width-1 {
			break
		}
		buf.SetCell(x, height-1, Cell{Rune: r, Style: d.buttonStyle})
	}
}

// HandleKey handles keyboard input.
func (d *Dialog) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		d.result = DialogOK
		if d.onOK != nil {
			d.onOK()
		}
		return true, nil

	case tea.KeyEsc:
		d.result = DialogCancel
		if d.onCancel != nil {
			d.onCancel()
		}
		return true, nil
	}

	// Pass other keys to content
	if d.content != nil {
		return d.content.HandleKey(msg)
	}

	return false, nil
}

// SetTitleStyle sets the title style.
func (d *Dialog) SetTitleStyle(style lipgloss.Style) {
	d.titleStyle = style
}

// SetButtonStyle sets the button label style.
func (d *Dialog) SetButtonStyle(style lipgloss.Style) {
	d.buttonStyle = style
}

// SetButtonHintStyle sets the button hint style (for Enter/Esc).
func (d *Dialog) SetButtonHintStyle(style lipgloss.Style) {
	d.buttonHintStyle = style
}
