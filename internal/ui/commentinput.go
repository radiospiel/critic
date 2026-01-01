package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommentInputMode represents the mode of the comment input
type CommentInputMode int

const (
	CommentModeHidden CommentInputMode = iota // Not showing comment input
	CommentModeNew                            // Creating a new comment
	CommentModeEdit                           // Editing an existing comment
)

// CommentInputModel represents the comment input component
type CommentInputModel struct {
	textarea     textarea.Model
	mode         CommentInputMode
	lineNumber   int    // Line number being commented on
	filePath     string // File path being commented on
	width        int
	height       int
	initialValue string // Initial value for edit mode
}

// NewCommentInputModel creates a new comment input model
func NewCommentInputModel() CommentInputModel {
	ta := textarea.New()
	ta.Placeholder = "Enter your review comment (markdown supported)...\nCtrl+S to save, Esc to cancel"
	ta.Focus()
	ta.CharLimit = 10000
	ta.ShowLineNumbers = false

	return CommentInputModel{
		textarea: ta,
		mode:     CommentModeHidden,
	}
}

// Init initializes the comment input model
func (m CommentInputModel) Init() tea.Cmd {
	return nil
}

// Update updates the comment input model
func (m CommentInputModel) Update(msg tea.Msg) (CommentInputModel, tea.Cmd) {
	var cmd tea.Cmd

	if m.mode == CommentModeHidden {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel comment input
			m.mode = CommentModeHidden
			m.textarea.Reset()
			return m, nil

		case "ctrl+s":
			// Save comment
			content := strings.TrimSpace(m.textarea.Value())
			if content != "" {
				// Send a message to save the comment
				return m, func() tea.Msg {
					return CommentSavedMsg{
						LineNumber: m.lineNumber,
						FilePath:   m.filePath,
						Content:    content,
					}
				}
			}
			return m, nil
		}
	}

	// Update textarea
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the comment input
func (m CommentInputModel) View() string {
	if m.mode == CommentModeHidden {
		return ""
	}

	// Create a bordered box with the textarea
	title := "Add Review Comment"
	if m.mode == CommentModeEdit {
		title = "Edit Review Comment"
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1).
		Width(m.width - 4).
		Height(m.height - 4)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#666"}).
		Padding(0, 1)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		"",
		m.textarea.View(),
		"",
		helpStyle.Render("Ctrl+S: Save • Esc: Cancel"),
	)

	return borderStyle.Render(content)
}

// Show shows the comment input for a new comment
func (m *CommentInputModel) Show(lineNumber int, filePath string) {
	m.mode = CommentModeNew
	m.lineNumber = lineNumber
	m.filePath = filePath
	m.textarea.Reset()
	m.textarea.Focus()
	m.initialValue = ""
}

// ShowEdit shows the comment input for editing an existing comment
func (m *CommentInputModel) ShowEdit(lineNumber int, filePath string, content string) {
	m.mode = CommentModeEdit
	m.lineNumber = lineNumber
	m.filePath = filePath
	m.textarea.SetValue(content)
	m.textarea.Focus()
	m.initialValue = content
}

// Hide hides the comment input
func (m *CommentInputModel) Hide() {
	m.mode = CommentModeHidden
	m.textarea.Reset()
}

// IsVisible returns whether the comment input is visible
func (m *CommentInputModel) IsVisible() bool {
	return m.mode != CommentModeHidden
}

// GetMode returns the current mode
func (m *CommentInputModel) GetMode() CommentInputMode {
	return m.mode
}

// SetSize sets the size of the comment input
func (m *CommentInputModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Set textarea size (accounting for borders and padding)
	m.textarea.SetWidth(width - 8)
	m.textarea.SetHeight(height - 10)
}

// CommentSavedMsg is sent when a comment is saved
type CommentSavedMsg struct {
	LineNumber int
	FilePath   string
	Content    string
}

// CommentDeletedMsg is sent when a comment is deleted
type CommentDeletedMsg struct {
	LineNumber int
	FilePath   string
}
