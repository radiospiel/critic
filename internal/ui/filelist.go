package ui

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/comments"
	"git.15b.it/eno/critic/internal/git"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileListModel represents the file list pane
type FileListModel struct {
	files          []*ctypes.FileDiff
	cursor         int
	viewport       viewport.Model
	width          int
	height         int
	ready          bool
	activeFile     *ctypes.FileDiff
	focused        bool
	commentManager *comments.FileManager
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
		style := normalFileStyle

		// Check if file has comments
		hasComments := false
		if m.commentManager != nil {
			// Use the git-relative path for checking comments
			gitPath := file.NewPath
			if file.IsDeleted {
				gitPath = file.OldPath
			}
			hasComments = m.commentManager.HasComments(gitPath)
		}

		// Apply styles based on cursor position
		if i == m.cursor {
			// Use active or inactive selection style based on focus
			if m.focused {
				style = selectedFileActiveStyle
			} else {
				style = selectedFileInactiveStyle
			}
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

		// Get file path to display (convert git-relative to cwd-relative)
		path := git.GitPathToDisplayPath(file.NewPath)
		if file.IsDeleted {
			path = git.GitPathToDisplayPath(file.OldPath)
		}

		// Add left indicator: yellow half-block for commented files, space for others
		var leftIndicator string
		if hasComments {
			const yellowBlock = "\x1b[38;5;220m▌\x1b[0m"
			leftIndicator = yellowBlock
		} else {
			leftIndicator = " "
		}

		// Build the content (status + path) that will be styled
		content := fmt.Sprintf("%s %s", status, path)

		// Render based on whether this is selected
		if i == m.cursor {
			// For selected line: render indicator first, then styled content spanning to right edge
			availableWidth := m.width - 1 // -1 for left indicator
			if availableWidth > 0 {
				// Apply selection style with full width spanning to right edge
				styledContent := style.Width(availableWidth).Render(content)
				b.WriteString(leftIndicator)
				b.WriteString(styledContent)
			} else {
				// Fallback if width is too small
				line := fmt.Sprintf("%s%s", leftIndicator, content)
				b.WriteString(style.Render(line))
			}
		} else {
			// For non-selected lines: render as before
			line := fmt.Sprintf("%s%s", leftIndicator, content)
			if m.width > 0 {
				style = style.MaxWidth(m.width).Inline(true)
			}
			b.WriteString(style.Render(line))
		}
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

// SetFocused sets whether this pane is focused
func (m *FileListModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetCommentManager sets the comment manager for checking file comments
func (m *FileListModel) SetCommentManager(cm *comments.FileManager) {
	m.commentManager = cm
}
