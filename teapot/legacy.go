package teapot

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	BaseWidget
	model   LegacyModel
	cached  string
}

// NewLegacyAdapter creates a new adapter wrapping a legacy model.
func NewLegacyAdapter(model LegacyModel) *LegacyAdapter {
	return &LegacyAdapter{
		BaseWidget: NewBaseWidget(),
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
	l.BaseWidget.SetBounds(bounds)
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
	l.cached = view

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
		x := 0
		for _, r := range line {
			if x >= buf.Width() {
				break
			}
			buf.SetCell(x, y, Cell{Rune: r})
			x++
		}
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

// StringWidget is a simple widget that displays pre-rendered string content.
// Useful for wrapping legacy View() output or external content.
type StringWidget struct {
	BaseWidget
	content string
}

// NewStringWidget creates a new string widget.
func NewStringWidget(content string) *StringWidget {
	return &StringWidget{
		BaseWidget: NewBaseWidget(),
		content:    content,
	}
}

// SetContent sets the string content.
func (s *StringWidget) SetContent(content string) {
	s.content = content
}

// Content returns the current content.
func (s *StringWidget) Content() string {
	return s.content
}

// Render renders the string content to the buffer.
func (s *StringWidget) Render(buf *SubBuffer) {
	lines := strings.Split(s.content, "\n")

	for y, line := range lines {
		if y >= buf.Height() {
			break
		}

		x := 0
		for _, r := range line {
			if x >= buf.Width() {
				break
			}
			buf.SetCell(x, y, Cell{Rune: r})
			x++
		}
	}
}

// CallbackWidget is a widget that uses a callback function for rendering.
// Useful for quick custom widgets without creating a new type.
type CallbackWidget struct {
	BaseWidget
	renderFn func(buf *SubBuffer, bounds Rect, focused bool)
	keyFn    func(msg tea.KeyMsg) (bool, tea.Cmd)
}

// NewCallbackWidget creates a new callback widget.
func NewCallbackWidget(renderFn func(buf *SubBuffer, bounds Rect, focused bool)) *CallbackWidget {
	return &CallbackWidget{
		BaseWidget: NewBaseWidget(),
		renderFn:   renderFn,
	}
}

// SetRenderFunc sets the render callback.
func (c *CallbackWidget) SetRenderFunc(fn func(buf *SubBuffer, bounds Rect, focused bool)) {
	c.renderFn = fn
}

// SetKeyFunc sets the key handling callback.
func (c *CallbackWidget) SetKeyFunc(fn func(msg tea.KeyMsg) (bool, tea.Cmd)) {
	c.keyFn = fn
}

// Render calls the render callback.
func (c *CallbackWidget) Render(buf *SubBuffer) {
	if c.renderFn != nil {
		c.renderFn(buf, c.bounds, c.focused)
	}
}

// HandleKey calls the key callback.
func (c *CallbackWidget) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if c.keyFn != nil {
		return c.keyFn(msg)
	}
	return false, nil
}
