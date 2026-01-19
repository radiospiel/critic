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

// ModalDialog is a modal dialog with OK and Cancel buttons.
// OK is triggered by Enter/Return, Cancel by Escape.
// When closed, it automatically clears itself from the focus manager's modal.
type ModalDialog struct {
	BaseView
	content      View
	title        string
	okLabel      string
	cancelLabel  string
	result       DialogResult
	focusManager *FocusManager // Reference to clear modal on close

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

// NewModalDialog creates a new modal dialog with the given content and title.
func NewModalDialog(content View, title string) *ModalDialog {
	d := &ModalDialog{
		BaseView:  NewBaseView(),
		content:     content,
		title:       title,
		okLabel:     "OK",
		cancelLabel: "Cancel",
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("220")), // Yellow
		buttonStyle: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#888"}),
		buttonOKStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")),
		buttonHintStyle: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666", Dark: "#888"}),
		borderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")), // Yellow
	}
	// Set up border using the new border system
	border := DoubleBorder().WithStyle(d.borderStyle)
	d.SetBorder(border)
	d.SetBorderTitle(title)
	d.updateFooter()

	if content != nil {
		content.SetParent(d)
	}
	return d
}

// SetTitle sets the dialog title.
func (d *ModalDialog) SetTitle(title string) {
	d.title = title
	d.SetBorderTitle(title)
}

// SetLabels sets the OK and Cancel button labels.
func (d *ModalDialog) SetLabels(ok, cancel string) {
	d.okLabel = ok
	d.cancelLabel = cancel
	d.updateFooter()
}

// updateFooter updates the border footer with button hints.
func (d *ModalDialog) updateFooter() {
	footer := "Enter: " + d.okLabel + " │ Esc: " + d.cancelLabel
	d.SetBorderFooter(footer)
}

// OnOK sets the callback for when OK is pressed.
func (d *ModalDialog) OnOK(fn func()) {
	d.onOK = fn
}

// OnCancel sets the callback for when Cancel is pressed.
func (d *ModalDialog) OnCancel(fn func()) {
	d.onCancel = fn
}

// Result returns the last dialog result.
func (d *ModalDialog) Result() DialogResult {
	return d.result
}

// ResetResult resets the dialog result to None.
func (d *ModalDialog) ResetResult() {
	d.result = DialogNone
}

// SetFocusManager sets the focus manager reference.
// This is called when the dialog is shown as a modal.
func (d *ModalDialog) SetFocusManager(fm *FocusManager) {
	d.focusManager = fm
}

// Close closes the modal dialog and clears it from the focus manager.
// This should be called when the dialog is dismissed (OK or Cancel).
func (d *ModalDialog) Close() {
	if d.focusManager != nil {
		d.focusManager.ClearModal()
	}
}

// Content returns the dialog's content widget.
func (d *ModalDialog) Content() View {
	return d.content
}

// SetContent sets the dialog's content widget.
func (d *ModalDialog) SetContent(w View) {
	if d.content != nil {
		d.content.SetParent(nil)
	}
	d.content = w
	if w != nil {
		w.SetParent(d)
	}
}

// Children returns the content widget.
func (d *ModalDialog) Children() []View {
	if d.content != nil {
		return []View{d.content}
	}
	return nil
}

// SetBounds sets the dialog bounds and layouts the content.
func (d *ModalDialog) SetBounds(bounds Rect) {
	d.BaseView.SetBounds(bounds)

	if d.content == nil {
		return
	}

	// Content gets the inner area (inside border)
	// The border rendering is handled by RenderView
	contentBounds := d.ContentBounds()
	d.content.SetBounds(contentBounds)
}

// Render renders the dialog content.
// The border, title, and footer are rendered by RenderWidget.
func (d *ModalDialog) Render(buf *SubBuffer) {
	// Render content - the buf is already the content area inside the border
	if d.content != nil {
		RenderWidget(d.content, buf)
	}
}

// HandleKey handles keyboard input.
func (d *ModalDialog) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
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
func (d *ModalDialog) SetTitleStyle(style lipgloss.Style) {
	d.titleStyle = style
}

// SetButtonStyle sets the button label style.
func (d *ModalDialog) SetButtonStyle(style lipgloss.Style) {
	d.buttonStyle = style
}

// SetButtonHintStyle sets the button hint style (for Enter/Esc).
func (d *ModalDialog) SetButtonHintStyle(style lipgloss.Style) {
	d.buttonHintStyle = style
}
