package ui

import (
	"fmt"
	"strings"

	ctypes "git.15b.it/eno/critic/pkg/types"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileListModel represents the file list pane
type FileListModel struct {
	files        []*ctypes.FileDiff
	cursor       int
	viewport     viewport.Model
	width        int
	height       int
	ready        bool
	activeFile   *ctypes.FileDiff
}

// NewFileListModel creates a new file list model
func NewFileListModel() FileListModel {
	return FileListModel{
		files:  []*ctypes.FileDiff{},
		cursor: 0,
	}
}

// Init initializes the file list model
func (m FileListModel) Init() tea.Cmd {
	return nil
}

// Update updates the file list model
func (m FileListModel) Update(msg tea.Msg) (FileListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateActiveFile()
			}

		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
				m.updateActiveFile()
			}

		case "g": // Go to top
			m.cursor = 0
			m.updateActiveFile()

		case "G": // Go to bottom
			if len(m.files) > 0 {
				m.cursor = len(m.files) - 1
				m.updateActiveFile()
			}
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-3)
			m.ready = true
		}
	}

	return m, nil
}

// View renders the file list
func (m FileListModel) View() string {
	if len(m.files) == 0 {
		return lipgloss.NewStyle().
			Padding(1, 2).
			Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#666"}).
			Render("No files changed")
	}

	var b strings.Builder

	for i, file := range m.files {
		cursor := " "
		style := normalFileStyle

		if i == m.cursor {
			cursor = "▸"
			style = selectedFileStyle
		}

		// Determine file status symbol
		var status string
		if file.IsNew {
			status = "+"
		} else if file.IsDeleted {
			status = "-"
		} else if file.IsRenamed {
			status = "→"
		} else {
			status = "M"
		}

		// Get file path to display
		path := file.NewPath
		if file.IsDeleted {
			path = file.OldPath
		}

		line := fmt.Sprintf("%s %s %s", cursor, status, path)
		b.WriteString(style.Render(line))
		if i < len(m.files)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// SetFiles updates the list of files
func (m *FileListModel) SetFiles(files []*ctypes.FileDiff) {
	m.files = files
	if len(files) > 0 && m.cursor >= len(files) {
		m.cursor = len(files) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.updateActiveFile()
}

// GetActiveFile returns the currently selected file
func (m *FileListModel) GetActiveFile() *ctypes.FileDiff {
	return m.activeFile
}

// SelectByPath selects a file by its path
func (m *FileListModel) SelectByPath(path string) bool {
	for i, file := range m.files {
		filePath := file.NewPath
		if filePath == "" {
			filePath = file.OldPath
		}
		if filePath == path {
			m.cursor = i
			m.updateActiveFile()
			return true
		}
	}
	return false
}

// updateActiveFile updates the active file based on cursor position
func (m *FileListModel) updateActiveFile() {
	if len(m.files) > 0 && m.cursor >= 0 && m.cursor < len(m.files) {
		m.activeFile = m.files[m.cursor]
	} else {
		m.activeFile = nil
	}
}

// SetSize sets the size of the file list pane
func (m *FileListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.ready {
		m.viewport.Width = width
		m.viewport.Height = height - 3 // Account for title and padding
	}
}
