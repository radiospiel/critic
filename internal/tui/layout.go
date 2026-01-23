package tui

import (
	"strings"

	pot "git.15b.it/eno/critic/teapot"
	"github.com/charmbracelet/lipgloss"
)

// Pane represents which pane is currently focused
type Pane int

const (
	FileListPane Pane = iota
	DiffViewPane
)

// LayoutView manages the split pane layout and focus traversal
type LayoutView struct {
	width        int
	height       int
	focusedPane  Pane
	splitRatio   float64   // Ratio of width for left pane (0.0 to 1.0)
	children     []pot.View // Child views in stable order
	focusedIndex int       // Index of currently focused child
}

// NewLayoutView creates a new layout model
func NewLayoutView() LayoutView {
	return LayoutView{
		focusedPane: FileListPane,
		splitRatio:  0.3, // 30% for file list, 70% for diff view
	}
}

// SetSize sets the overall size of the layout
func (m *LayoutView) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// GetFileListSize returns the size for the file list pane
func (m *LayoutView) GetFileListSize() (int, int) {
	leftWidth := int(float64(m.width) * m.splitRatio)
	// Account for separator (1 char)
	if leftWidth > 0 {
		leftWidth -= 1
	}
	height := m.height - 1 // Account for status bar
	return leftWidth, height
}

// GetDiffViewSize returns the size for the diff view pane
func (m *LayoutView) GetDiffViewSize() (int, int) {
	leftWidth := int(float64(m.width) * m.splitRatio)
	rightWidth := m.width - leftWidth - 1 // -1 for separator
	height := m.height - 1 // Account for status bar
	return rightWidth, height
}

// GetFocusedPane returns the currently focused pane
func (m *LayoutView) GetFocusedPane() Pane {
	return m.focusedPane
}

// SetFocusedPane sets which pane is focused
func (m *LayoutView) SetFocusedPane(pane Pane) {
	m.focusedPane = pane
}

// SetChildren sets the child views in stable order for focus traversal
func (m *LayoutView) SetChildren(children ...pot.View) {
	m.children = children
	m.focusedIndex = -1 // Reset focused index
	// Find first focusable child
	for i, child := range children {
		if child.AcceptsFocus() {
			m.focusedIndex = i
			break
		}
	}
}

// Children returns the child views in stable order
func (m *LayoutView) Children() []pot.View {
	return m.children
}

// FocusNext moves focus to the next focusable child view.
// Returns true if focus was successfully transferred to a different child,
// false if there are no focusable children or only one focusable child.
func (m *LayoutView) FocusNext() bool {
	return m.moveFocus(1)
}

// FocusPrev moves focus to the previous focusable child view.
// Returns true if focus was successfully transferred to a different child,
// false if there are no focusable children or only one focusable child.
func (m *LayoutView) FocusPrev() bool {
	return m.moveFocus(-1)
}

// moveFocus moves focus by the given delta (positive for next, negative for prev).
// Returns false if there is no next/previous focusable child in the requested direction.
func (m *LayoutView) moveFocus(delta int) bool {
	if len(m.children) == 0 {
		return false
	}

	// Build list of focusable children
	var focusableIndices []int
	for i, child := range m.children {
		if child.AcceptsFocus() {
			focusableIndices = append(focusableIndices, i)
		}
	}

	// Need at least 2 focusable children to move
	if len(focusableIndices) < 2 {
		return false
	}

	// Find current position in focusable list
	currentPos := -1
	for i, idx := range focusableIndices {
		if idx == m.focusedIndex {
			currentPos = i
			break
		}
	}

	// Calculate new position, return false if out of bounds
	newPos := currentPos + delta
	if newPos < 0 || newPos >= len(focusableIndices) {
		return false
	}
	newIndex := focusableIndices[newPos]

	// Update focus state
	if m.focusedIndex >= 0 && m.focusedIndex < len(m.children) {
		m.children[m.focusedIndex].SetFocused(false)
	}
	m.focusedIndex = newIndex
	m.children[m.focusedIndex].SetFocused(true)

	// Update legacy focusedPane for compatibility
	if m.focusedIndex == 0 {
		m.focusedPane = FileListPane
	} else if m.focusedIndex == 1 {
		m.focusedPane = DiffViewPane
	}

	return true
}

// GetFocusedChild returns the currently focused child view, or nil if none
func (m *LayoutView) GetFocusedChild() pot.View {
	if m.focusedIndex >= 0 && m.focusedIndex < len(m.children) {
		return m.children[m.focusedIndex]
	}
	return nil
}

// SetFocusedChild sets focus to the specified child view
// Returns true if the child was found and can accept focus
func (m *LayoutView) SetFocusedChild(child pot.View) bool {
	for i, c := range m.children {
		if c == child && c.AcceptsFocus() {
			if m.focusedIndex >= 0 && m.focusedIndex < len(m.children) {
				m.children[m.focusedIndex].SetFocused(false)
			}
			m.focusedIndex = i
			m.children[i].SetFocused(true)

			// Update legacy focusedPane for compatibility
			if i == 0 {
				m.focusedPane = FileListPane
			} else if i == 1 {
				m.focusedPane = DiffViewPane
			}
			return true
		}
	}
	return false
}

// RenderSplitView renders two panes side-by-side
func (m *LayoutView) RenderSplitView(leftContent, rightContent string) string {
	leftWidth, leftHeight := m.GetFileListSize()
	rightWidth, _ := m.GetDiffViewSize()

	// No dimming - selection highlighting handles focus indication
	leftStyle := lipgloss.NewStyle().Width(leftWidth).Height(leftHeight)
	rightStyle := lipgloss.NewStyle().Width(rightWidth).Height(leftHeight)

	leftView := leftStyle.Render(leftContent)
	rightView := rightStyle.Render(rightContent)

	// Create separator - repeat │ for full height
	var sepBuilder strings.Builder
	for i := 0; i < leftHeight; i++ {
		sepBuilder.WriteString("│")
		if i < leftHeight-1 {
			sepBuilder.WriteString("\n")
		}
	}
	separator := sepBuilder.String()

	// Join: left + separator + right
	return lipgloss.JoinHorizontal(lipgloss.Top, leftView, separator, rightView)
}
