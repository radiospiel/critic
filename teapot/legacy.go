package teapot

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LegacyModel is the interface that existing BubbleTea components should implement.
// This allows them to be wrapped as widgets during the migration period.
type LegacyModel interface {
	// Update handles BubbleTea messages.
	Update(msg tea.Msg) (LegacyModel, tea.Cmd)
	// View returns the rendered string.
	View() string
	// SetSize sets the component's size.
	SetSize(width, height int)
}

// LegacyAdapter wraps a LegacyModel as a Widget.
// This enables incremental migration from string-based rendering to the widget system.
type LegacyAdapter struct {
	BaseView
	model LegacyModel
}

// NewLegacyAdapter creates a new adapter wrapping a legacy model.
func NewLegacyAdapter(model LegacyModel) *LegacyAdapter {
	return &LegacyAdapter{
		BaseView: NewBaseView(),
		model:      model,
	}
}

// Model returns the wrapped legacy model.
func (l *LegacyAdapter) Model() LegacyModel {
	return l.model
}

// SetModel sets the wrapped legacy model.
func (l *LegacyAdapter) SetModel(model LegacyModel) {
	l.model = model
	if model != nil {
		model.SetSize(l.bounds.Width, l.bounds.Height)
	}
}

// SetBounds sets the adapter's bounds and propagates to the model.
func (l *LegacyAdapter) SetBounds(bounds Rect) {
	l.BaseView.SetBounds(bounds)
	if l.model != nil {
		l.model.SetSize(bounds.Width, bounds.Height)
	}
}

// Render renders the legacy model's View() output to the buffer.
func (l *LegacyAdapter) Render(buf *SubBuffer) {
	if l.model == nil {
		return
	}

	// Get the string output from the legacy model
	view := l.model.View()

	// Parse the string output and render to buffer
	// This handles basic ANSI sequences and newlines
	l.renderString(buf, view)
}

// renderString renders a string with basic ANSI handling to the buffer.
func (l *LegacyAdapter) renderString(buf *SubBuffer, s string) {
	lines := strings.Split(s, "\n")

	for y, line := range lines {
		if y >= buf.Height() {
			break
		}

		// For now, just render the raw characters
		// In a full implementation, we'd parse ANSI escape sequences
		buf.SetString(0, y, line, noStyle)
	}
}

// HandleKey passes key events to the legacy model.
func (l *LegacyAdapter) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if l.model == nil {
		return false, nil
	}

	newModel, cmd := l.model.Update(msg)
	l.model = newModel
	return true, cmd
}

// HandleMouse passes mouse events to the legacy model.
func (l *LegacyAdapter) HandleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	if l.model == nil {
		return false, nil
	}

	newModel, cmd := l.model.Update(msg)
	l.model = newModel
	return true, cmd
}

// Update passes any message to the legacy model.
func (l *LegacyAdapter) Update(msg tea.Msg) tea.Cmd {
	if l.model == nil {
		return nil
	}

	newModel, cmd := l.model.Update(msg)
	l.model = newModel
	return cmd
}

// StringView is a simple widget that displays pre-rendered string content.
// Useful for wrapping legacy View() output or external content.
type StringView struct {
	BaseView
	content string
}

// NewStringView creates a new string widget.
func NewStringView(content string) *StringView {
	return &StringView{
		BaseView: NewBaseView(),
		content:    content,
	}
}

// SetContent sets the string content.
func (s *StringView) SetContent(content string) {
	s.content = content
}

// Content returns the current content.
func (s *StringView) Content() string {
	return s.content
}

// Render renders the string content to the buffer.
func (s *StringView) Render(buf *SubBuffer) {
	lines := strings.Split(s.content, "\n")

	for y, line := range lines {
		if y >= buf.Height() {
			break
		}
		buf.SetString(0, y, line, noStyle)
	}
}

// CallbackView is a widget that uses a callback function for rendering.
// Useful for quick custom widgets without creating a new type.
type CallbackView struct {
	BaseView
	renderFn func(buf *SubBuffer, bounds Rect, focused bool)
	keyFn    func(msg tea.KeyMsg) (bool, tea.Cmd)
}

// NewCallbackView creates a new callback widget.
func NewCallbackView(renderFn func(buf *SubBuffer, bounds Rect, focused bool)) *CallbackView {
	return &CallbackView{
		BaseView: NewBaseView(),
		renderFn:   renderFn,
	}
}

// SetRenderFunc sets the render callback.
func (c *CallbackView) SetRenderFunc(fn func(buf *SubBuffer, bounds Rect, focused bool)) {
	c.renderFn = fn
}

// SetKeyFunc sets the key handling callback.
func (c *CallbackView) SetKeyFunc(fn func(msg tea.KeyMsg) (bool, tea.Cmd)) {
	c.keyFn = fn
}

// Render calls the render callback.
func (c *CallbackView) Render(buf *SubBuffer) {
	if c.renderFn != nil {
		c.renderFn(buf, c.bounds, c.focused)
	}
}

// HandleKey calls the key callback.
func (c *CallbackView) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if c.keyFn != nil {
		return c.keyFn(msg)
	}
	return false, nil
}

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

// Focusable returns true as the textarea can receive focus.
func (t *TextAreaView) Focusable() bool {
	return true
}

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
