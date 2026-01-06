package ui

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/config"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/pkg/critic"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileListModel represents the file list pane
type FileListModel struct {
	files        []*ctypes.FileDiff
	cursor       int
	scrollOffset int // First visible file index
	viewport     viewport.Model
	width        int
	height       int
	ready        bool
	activeFile   *ctypes.FileDiff
	focused      bool
	messaging    critic.Messaging
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
				m.ensureCursorVisible()
			}

		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
				m.updateActiveFile()
				m.ensureCursorVisible()
			}

		case "shift+up":
			if m.cursor > 0 {
				m.cursor -= config.ShiftArrowJumpSize
				if m.cursor < 0 {
					m.cursor = 0
				}
				m.updateActiveFile()
				m.ensureCursorVisible()
			}

		case "shift+down":
			if m.cursor < len(m.files)-1 {
				m.cursor += config.ShiftArrowJumpSize
				if m.cursor >= len(m.files) {
					m.cursor = len(m.files) - 1
				}
				m.updateActiveFile()
				m.ensureCursorVisible()
			}

		case "g": // Go to top
			m.cursor = 0
			m.updateActiveFile()
			m.ensureCursorVisible()

		case "G": // Go to bottom
			if len(m.files) > 0 {
				m.cursor = len(m.files) - 1
				m.updateActiveFile()
				m.ensureCursorVisible()
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

// ensureCursorVisible scrolls the view to keep the cursor visible
func (m *FileListModel) ensureCursorVisible() {
	if m.height <= 0 {
		return
	}
	// If cursor is above the visible area, scroll up
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	// If cursor is below the visible area, scroll down
	if m.cursor >= m.scrollOffset+m.height {
		m.scrollOffset = m.cursor - m.height + 1
	}
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

	// Calculate visible range based on scrollOffset
	startIdx := m.scrollOffset
	endIdx := startIdx + m.height
	if endIdx > len(m.files) {
		endIdx = len(m.files)
	}

	for i := startIdx; i < endIdx; i++ {
		file := m.files[i]
		style := normalFileStyle

		// Get the git-relative path for checking conversations
		gitPath := file.NewPath
		if file.IsDeleted {
			gitPath = file.OldPath
		}

		// Get conversation summary from messaging interface
		var hasUnreadAI bool
		var hasUnresolved bool
		var hasResolved bool

		if m.messaging != nil {
			summary, err := m.messaging.GetFileConversationSummary(gitPath)
			if err == nil && summary != nil {
				hasUnreadAI = summary.HasUnreadAIMessages
				hasUnresolved = summary.HasUnresolvedComments
				hasResolved = summary.HasResolvedComments
			}
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

		// Add left indicator:
		// - Red/bright block for files with unread AI comments
		// - Yellow block for files with unresolved comments
		// - Green block for files with only resolved comments
		// - Space for files without comments
		var leftIndicator string
		if hasUnreadAI {
			// Red block for unread AI comments (most attention-grabbing)
			const redBlock = "\x1b[38;5;196m▌\x1b[0m"
			leftIndicator = redBlock
		} else if hasUnresolved {
			// Yellow block for unresolved comments
			const yellowBlock = "\x1b[38;5;220m▌\x1b[0m"
			leftIndicator = yellowBlock
		} else if hasResolved {
			// Green block for resolved comments
			const greenBlock = "\x1b[38;5;34m▌\x1b[0m"
			leftIndicator = greenBlock
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
				// Truncate content if it's too long, then apply width to span to edge
				if lipgloss.Width(content) > availableWidth {
					// Truncate to fit
					runes := []rune(content)
					if len(runes) > availableWidth {
						content = string(runes[:availableWidth])
					}
				}
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
		if i < endIdx-1 {
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
	m.ensureCursorVisible()
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
			m.ensureCursorVisible()
			return true
		}
	}
	return false
}

// SelectNext moves to the next file in the list. Returns true if moved.
func (m *FileListModel) SelectNext() bool {
	if m.cursor < len(m.files)-1 {
		m.cursor++
		m.updateActiveFile()
		m.ensureCursorVisible()
		return true
	}
	return false
}

// SelectPrev moves to the previous file in the list. Returns true if moved.
func (m *FileListModel) SelectPrev() bool {
	if m.cursor > 0 {
		m.cursor--
		m.updateActiveFile()
		m.ensureCursorVisible()
		return true
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

// SetMessaging sets the messaging interface for checking conversation status
func (m *FileListModel) SetMessaging(messaging critic.Messaging) {
	m.messaging = messaging
}
