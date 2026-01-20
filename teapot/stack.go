package teapot

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Stack is a container that layers widgets on top of each other.
// The last widget in the stack is rendered on top and receives input first.
// Useful for modal dialogs, overlays, and popups.
type Stack struct {
	ContainerView
}

// NewStack creates a new stack container.
func NewStack() *Stack {
	s := &Stack{
		ContainerView: NewContainerView(),
	}
	s.SetFocusable(false)
	return s
}

// Push adds a widget to the top of the stack.
func (s *Stack) Push(w View) {
	s.AddChild(w)
	// Give the new widget the same bounds as the stack
	w.SetBounds(s.bounds)
	s.Repaint() // Stack needs repainting
}

// Pop removes and returns the top widget from the stack.
func (s *Stack) Pop() View {
	if len(s.children) == 0 {
		return nil
	}
	top := s.children[len(s.children)-1]
	s.RemoveChild(top)
	s.Repaint() // Stack needs repainting
	return top
}

// Top returns the top widget without removing it.
func (s *Stack) Top() View {
	if len(s.children) == 0 {
		return nil
	}
	return s.children[len(s.children)-1]
}

// Base returns the bottom (first) widget.
func (s *Stack) Base() View {
	if len(s.children) == 0 {
		return nil
	}
	return s.children[0]
}

// SetBounds sets the bounds and propagates to all children.
func (s *Stack) SetBounds(bounds Rect) {
	s.bounds = bounds
	for _, child := range s.children {
		child.SetBounds(bounds)
	}
}

// Render renders all layers from bottom to top.
func (s *Stack) Render(buf *SubBuffer) {
	for _, child := range s.children {
		RenderWidget(child, buf)
	}
}

// HandleKey routes key events to the top widget first.
func (s *Stack) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Top widget gets first chance to handle input
	for i := len(s.children) - 1; i >= 0; i-- {
		child := s.children[i]
		if handled, cmd := child.HandleKey(msg); handled {
			return handled, cmd
		}
	}
	return false, nil
}

// HandleMouse routes mouse events to the top widget first.
func (s *Stack) HandleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	for i := len(s.children) - 1; i >= 0; i-- {
		child := s.children[i]
		if handled, cmd := child.HandleMouse(msg); handled {
			return handled, cmd
		}
	}
	return false, nil
}

// Modal is a widget that displays a dialog box over other content.
// It captures all input and prevents interaction with content below.
type Modal struct {
	BaseView
	content       View
	title         string
	width         int  // 0 = auto
	height        int  // 0 = auto
	centerH       bool // Center horizontally
	centerV       bool // Center vertically
	showBorder    bool
	borderStyle   lipgloss.Style
	bgStyle       lipgloss.Style
	dimBackground bool
}

// NewModal creates a new modal dialog with the given content.
func NewModal(content View, title string) *Modal {
	m := &Modal{
		BaseView:      NewBaseView(),
		content:       content,
		title:         title,
		centerH:       true,
		centerV:       true,
		showBorder:    true,
		borderStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		bgStyle:       lipgloss.NewStyle().Background(lipgloss.Color("236")),
		dimBackground: true,
	}
	if content != nil {
		content.SetParent(m)
	}
	return m
}

// SetSize sets a fixed size for the modal.
func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.Repaint()
}

// SetCentered sets whether the modal is centered.
func (m *Modal) SetCentered(horizontal, vertical bool) {
	m.centerH = horizontal
	m.centerV = vertical
	m.Repaint()
}

// SetDimBackground sets whether to dim the background.
func (m *Modal) SetDimBackground(dim bool) {
	m.dimBackground = dim
	m.Repaint()
}

// SetBorderStyle sets the border style.
func (m *Modal) SetBorderStyle(style lipgloss.Style) {
	m.borderStyle = style
	m.Repaint()
}

// SetBackgroundStyle sets the background style inside the modal.
func (m *Modal) SetBackgroundStyle(style lipgloss.Style) {
	m.bgStyle = style
	m.Repaint()
}

// Children returns the modal's content widget.
func (m *Modal) Children() []View {
	if m.content != nil {
		return []View{m.content}
	}
	return nil
}

// SetBounds sets the modal's bounds and calculates content position.
func (m *Modal) SetBounds(bounds Rect) {
	m.BaseView.SetBounds(bounds)

	if m.content == nil {
		return
	}

	// Calculate modal size
	modalWidth := m.width
	modalHeight := m.height

	if modalWidth == 0 {
		c := m.content.Constraints()
		modalWidth = c.EffectivePreferredWidth()
		if modalWidth == 0 {
			modalWidth = bounds.Width / 2
		}
		modalWidth += 2 // Border
	}

	if modalHeight == 0 {
		c := m.content.Constraints()
		modalHeight = c.EffectivePreferredHeight()
		if modalHeight == 0 {
			modalHeight = bounds.Height / 2
		}
		modalHeight += 2 // Border
		if m.title != "" {
			modalHeight++ // Title takes no extra space with border
		}
	}

	// Clamp to available space
	modalWidth = min(modalWidth, bounds.Width)
	modalHeight = min(modalHeight, bounds.Height)

	// Calculate position
	var x, y int
	if m.centerH {
		x = bounds.X + (bounds.Width-modalWidth)/2
	} else {
		x = bounds.X
	}
	if m.centerV {
		y = bounds.Y + (bounds.Height-modalHeight)/2
	} else {
		y = bounds.Y
	}

	// Set content bounds (inside border)
	contentX := x + 1
	contentY := y + 1
	contentWidth := modalWidth - 2
	contentHeight := modalHeight - 2

	m.content.SetBounds(Rect{
		Position{X: contentX, Y: contentY},
		Size{Width: contentWidth, Height: contentHeight},
	})
}

// Render renders the modal with optional dimmed background.
func (m *Modal) Render(buf *SubBuffer) {
	// Dim background if enabled
	if m.dimBackground && buf.Width() > 0 {
		dimStyle := lipgloss.NewStyle().Faint(true)
		// y values from 0 to Height()-1 are valid, cells is full width (non-empty)
		for y := 0; y < buf.Height(); y++ {
			cells := make([]Cell, buf.Width())
			for x := 0; x < buf.Width(); x++ {
				cells[x] = buf.GetCell(x, y)
				cells[x].Style = dimStyle
			}
			buf.setCells(0, y, cells)
		}
	}

	if m.content == nil {
		return
	}

	// Calculate modal bounds
	contentBounds := m.content.Bounds()
	modalRect := Rect{
		Position{
			X: contentBounds.X - m.bounds.X - 1,
			Y: contentBounds.Y - m.bounds.Y - 1,
		},
		Size{
			Width:  contentBounds.Width + 2,
			Height: contentBounds.Height + 2,
		},
	}

	// Fill background
	bgRow := make([]Cell, modalRect.Width)
	for i := range bgRow {
		bgRow[i] = Cell{Rune: ' ', Style: m.bgStyle}
	}
	for y := modalRect.Y; y < modalRect.Y+modalRect.Height; y++ {
		buf.SetCells(modalRect.X, y, bgRow)
	}

	// Draw border
	if m.showBorder {
		innerWidth := modalRect.Width - 2

		// Top edge: ┌───┐
		topEdge := "┌" + strings.Repeat("─", innerWidth) + "┐"
		buf.SetString(modalRect.X, modalRect.Y, topEdge, m.borderStyle)

		// Bottom edge: └───┘
		bottomEdge := "└" + strings.Repeat("─", innerWidth) + "┘"
		buf.SetString(modalRect.X, modalRect.Y+modalRect.Height-1, bottomEdge, m.borderStyle)

		// Left and right edges
		for y := modalRect.Y + 1; y < modalRect.Y+modalRect.Height-1; y++ {
			buf.SetString(modalRect.X, y, "│", m.borderStyle)
			buf.SetString(modalRect.X+modalRect.Width-1, y, "│", m.borderStyle)
		}

		// Title
		if m.title != "" {
			title := " " + m.title + " "
			titleX := modalRect.X + (modalRect.Width-len(title))/2
			buf.SetString(titleX, modalRect.Y, title, m.borderStyle.Bold(true))
		}
	}

	// Render content
	contentSub := NewSubBuffer(buf.parent, NewRect(
		buf.offset.X+contentBounds.X-m.bounds.X,
		buf.offset.Y+contentBounds.Y-m.bounds.Y,
		contentBounds.Width,
		contentBounds.Height,
	))
	RenderWidget(m.content, contentSub)
}

// HandleKey passes events to the content and always returns true (modal captures all input).
func (m *Modal) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.content != nil {
		_, cmd := m.content.HandleKey(msg)
		return true, cmd // Always capture
	}
	return true, nil
}

// HandleMouse passes events to the content and always returns true.
func (m *Modal) HandleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	if m.content != nil {
		_, cmd := m.content.HandleMouse(msg)
		return true, cmd
	}
	return true, nil
}

// Content returns the modal's content widget.
func (m *Modal) Content() View {
	return m.content
}

// SetContent sets the modal's content widget.
func (m *Modal) SetContent(w View) {
	if m.content != nil {
		m.content.SetParent(nil)
	}
	m.content = w
	if w != nil {
		w.SetParent(m)
	}
	m.Repaint()
}
