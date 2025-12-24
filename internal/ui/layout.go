package ui

import (
	"strings"

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
	// Account for separator (1 char)
	if leftWidth > 0 {
		leftWidth -= 1
	}
	height := m.height - 1 // Account for status bar
	return leftWidth, height
}

// GetDiffViewSize returns the size for the diff view pane
func (m *LayoutModel) GetDiffViewSize() (int, int) {
	leftWidth := int(float64(m.width) * m.splitRatio)
	rightWidth := m.width - leftWidth - 1 // -1 for separator
	height := m.height - 1 // Account for status bar
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
