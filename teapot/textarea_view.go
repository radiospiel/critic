package teapot

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// TextAreaView wraps a bubbles textarea.Model as a View.
// This provides a proper widget interface for the textarea component.
type TextAreaView struct {
	BaseView
	model       textarea.Model
	constraints Constraints
}

// NewTextAreaView creates a new TextAreaView wrapping a textarea.Model.
func NewTextAreaView(model textarea.Model) *TextAreaView {
	return &TextAreaView{
		BaseView: NewBaseView(),
		model:    model,
	}
}

// Model returns the wrapped textarea.Model.
func (t *TextAreaView) Model() textarea.Model {
	return t.model
}

// SetModel sets the wrapped textarea.Model.
func (t *TextAreaView) SetModel(model textarea.Model) {
	t.model = model
	t.model.SetWidth(t.bounds.Width)
	t.model.SetHeight(t.bounds.Height)
}

// SetConstraints sets the layout constraints for the textarea.
func (t *TextAreaView) SetConstraints(c Constraints) {
	t.constraints = c
}

// Constraints returns the textarea's layout constraints.
func (t *TextAreaView) Constraints() Constraints {
	return t.constraints
}

// SetBounds sets the view bounds and propagates to the textarea model.
func (t *TextAreaView) SetBounds(bounds Rect) {
	t.BaseView.SetBounds(bounds)
	t.model.SetWidth(bounds.Width)
	t.model.SetHeight(bounds.Height)
}

// Render renders the textarea's View() output to the buffer.
func (t *TextAreaView) Render(buf *SubBuffer) {
	view := t.model.View()
	lines := strings.Split(view, "\n")

	for y, line := range lines {
		if y >= buf.Height() {
			break
		}

		parsedCells := ParseANSILine(line)

		// Build row with padding
		rowCells := make([]Cell, buf.Width())
		for x := range buf.Width() {
			if x < len(parsedCells) {
				rowCells[x] = parsedCells[x]
			} else {
				rowCells[x] = Cell{Rune: ' '}
			}
		}
		buf.SetCells(0, y, rowCells)
	}
}

// HandleKey passes key events to the textarea model.
func (t *TextAreaView) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return true, cmd
}

// HandleMouse passes mouse events to the textarea model.
func (t *TextAreaView) HandleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return true, cmd
}

// Focus activates the textarea.
func (t *TextAreaView) Focus() tea.Cmd {
	return t.model.Focus()
}

// Blur deactivates the textarea.
func (t *TextAreaView) Blur() {
	t.model.Blur()
}

// Value returns the textarea content.
func (t *TextAreaView) Value() string {
	return t.model.Value()
}

// SetValue sets the textarea content.
func (t *TextAreaView) SetValue(s string) {
	t.model.SetValue(s)
}

// AcceptsFocus returns true as the textarea can receive focus.
func (t *TextAreaView) AcceptsFocus() bool {
	return true
}

// FocusNext is a no-op for textarea as it has no focusable children.
func (t *TextAreaView) FocusNext() bool {
	return false
}

// FocusPrev is a no-op for textarea as it has no focusable children.
func (t *TextAreaView) FocusPrev() bool {
	return false
}
