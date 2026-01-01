package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommentEditor represents the comment editing UI
type CommentEditor struct {
	textarea textarea.Model
	active   bool
	lineNum  int
	width    int
	height   int
}

// NewCommentEditor creates a new comment editor
func NewCommentEditor() CommentEditor {
	ta := textarea.New()
	ta.Placeholder = "Enter your comment here..."
	ta.CharLimit = 10000
	ta.ShowLineNumbers = false

	return CommentEditor{
		textarea: ta,
		active:   false,
	}
}

// Init initializes the comment editor
func (m CommentEditor) Init() tea.Cmd {
	return nil
}

// Update handles messages for the comment editor
func (m *CommentEditor) Update(msg tea.Msg) tea.Cmd {
	if !m.active {
		return nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			// Save and continue editing
			return m.saveComment(false)

		case "ctrl+x":
			// Save and exit
			return m.saveComment(true)

		case "esc":
			// Cancel editing
			return m.cancelEdit()

		default:
			// Pass other keys to textarea
			m.textarea, cmd = m.textarea.Update(msg)
		}
	}

	return cmd
}

// View renders the comment editor
func (m CommentEditor) View() string {
	if !m.active {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(0, 1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#666"}).
		Padding(0, 1)

	title := titleStyle.Render("Edit Comment")
	help := helpStyle.Render("Ctrl+S: Save | Ctrl+X: Save & Exit | Esc: Cancel")

	content := m.textarea.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		help,
	)
}

// Activate activates the comment editor for a specific line
func (m *CommentEditor) Activate(lineNum int, existingComment string) tea.Cmd {
	m.active = true
	m.lineNum = lineNum
	m.textarea.SetValue(existingComment)
	m.textarea.Focus()
	return textarea.Blink
}

// Deactivate deactivates the comment editor
func (m *CommentEditor) Deactivate() {
	m.active = false
	m.textarea.Blur()
	m.textarea.SetValue("")
}

// IsActive returns whether the editor is active
func (m CommentEditor) IsActive() bool {
	return m.active
}

// GetLineNum returns the line number being edited
func (m CommentEditor) GetLineNum() int {
	return m.lineNum
}

// GetComment returns the current comment text
func (m CommentEditor) GetComment() string {
	return strings.TrimSpace(m.textarea.Value())
}

// SetSize sets the size of the comment editor
func (m *CommentEditor) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Reserve space for title and help text
	m.textarea.SetWidth(width - 4)
	m.textarea.SetHeight(height - 4)
}

// CommentSavedMsg is sent when a comment is saved
type CommentSavedMsg struct {
	LineNum int
	Comment string
	Exit    bool
}

// CommentCancelledMsg is sent when comment editing is cancelled
type CommentCancelledMsg struct{}

// saveComment saves the current comment
func (m *CommentEditor) saveComment(exit bool) tea.Cmd {
	comment := m.GetComment()
	lineNum := m.lineNum

	if exit {
		m.Deactivate()
	}

	return func() tea.Msg {
		return CommentSavedMsg{
			LineNum: lineNum,
			Comment: comment,
			Exit:    exit,
		}
	}
}

// cancelEdit cancels the current edit
func (m *CommentEditor) cancelEdit() tea.Cmd {
	m.Deactivate()
	return func() tea.Msg {
		return CommentCancelledMsg{}
	}
}
