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

	// Content gets the inner area (minus title and button row)
	// Title: 1 line, Buttons: 1 line, padding: 1 line each side
	contentBounds := Rect{
		X:      bounds.X,
		Y:      bounds.Y + 2, // After title + spacing
		Width:  bounds.Width,
		Height: bounds.Height - 4, // Title, spacing, content, buttons
	}
	d.content.SetBounds(contentBounds)
}

// Render renders the dialog.
func (d *Dialog) Render(buf *SubBuffer) {
	// Render title
	titleStr := d.titleStyle.Render(d.title)
	buf.SetString(1, 0, titleStr, lipgloss.NewStyle())

	// Render content
	if d.content != nil {
		contentBounds := d.content.Bounds()
		contentSub := buf.parent.Sub(Rect{
			X:      buf.offset.X,
			Y:      buf.offset.Y + 2,
			Width:  contentBounds.Width,
			Height: contentBounds.Height,
		})
		d.content.Render(contentSub)
	}

	// Render button hints at bottom
	buttonY := buf.Height() - 1
	hint := d.buttonHintStyle.Render("Enter") + d.buttonStyle.Render(": "+d.okLabel+"  ") +
		d.buttonHintStyle.Render("Esc") + d.buttonStyle.Render(": "+d.cancelLabel)

	buf.SetString(1, buttonY, hint, lipgloss.NewStyle())
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
