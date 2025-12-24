package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Pane represents which pane is currently focused
type Pane int

const (
	FileListPane Pane = iota
	DiffViewPane
)

// LayoutModel manages the split pane layout
type LayoutModel struct {
	width      int
	height     int
	focusedPane Pane
	splitRatio float64 // Ratio of width for left pane (0.0 to 1.0)
}

// NewLayoutModel creates a new layout model
func NewLayoutModel() LayoutModel {
	return LayoutModel{
		focusedPane: FileListPane,
		splitRatio:  0.3, // 30% for file list, 70% for diff view
	}
}

// SetSize sets the overall size of the layout
func (m *LayoutModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// GetFileListSize returns the size for the file list pane
func (m *LayoutModel) GetFileListSize() (int, int) {
	leftWidth := int(float64(m.width) * m.splitRatio)
	// Account for borders (2 chars per border)
	if leftWidth > 4 {
		leftWidth -= 4
	}
	height := m.height
	if height > 4 {
		height -= 4
	}
	return leftWidth, height
}

// GetDiffViewSize returns the size for the diff view pane
func (m *LayoutModel) GetDiffViewSize() (int, int) {
	leftWidth := int(float64(m.width) * m.splitRatio)
	rightWidth := m.width - leftWidth - 1 // -1 for spacing between panes
	// Account for borders
	if rightWidth > 4 {
		rightWidth -= 4
	}
	height := m.height
	if height > 4 {
		height -= 4
	}
	return rightWidth, height
}

// ToggleFocus switches focus between panes
func (m *LayoutModel) ToggleFocus() {
	if m.focusedPane == FileListPane {
		m.focusedPane = DiffViewPane
	} else {
		m.focusedPane = FileListPane
	}
}

// GetFocusedPane returns the currently focused pane
func (m *LayoutModel) GetFocusedPane() Pane {
	return m.focusedPane
}

// SetFocusedPane sets which pane is focused
func (m *LayoutModel) SetFocusedPane(pane Pane) {
	m.focusedPane = pane
}

// RenderSplitView renders two panes side-by-side
func (m *LayoutModel) RenderSplitView(leftContent, rightContent string) string {
	leftWidth, _ := m.GetFileListSize()
	rightWidth, _ := m.GetDiffViewSize()

	// Create bordered views
	leftStyle := inactiveBorderStyle.Width(leftWidth).Height(m.height - 2)
	rightStyle := inactiveBorderStyle.Width(rightWidth).Height(m.height - 2)

	if m.focusedPane == FileListPane {
		leftStyle = activeBorderStyle.Width(leftWidth).Height(m.height - 2)
	} else {
		rightStyle = activeBorderStyle.Width(rightWidth).Height(m.height - 2)
	}

	// Add titles
	leftView := leftStyle.Render(RenderTitle("Files", m.focusedPane == FileListPane) + "\n" + leftContent)
	rightView := rightStyle.Render(RenderTitle("Diff", m.focusedPane == DiffViewPane) + "\n" + rightContent)

	// Join side by side
	return lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)
}
