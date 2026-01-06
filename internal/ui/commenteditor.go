package ui

import (
	"strings"

	pot "git.15b.it/eno/critic/teapot"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// CommentEditor represents the comment editing UI using a teapot Dialog.
type CommentEditor struct {
	dialog   *pot.Dialog
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

	// Create a wrapper widget for the textarea
	textWidget := &textareaWidget{textarea: ta}

	dialog := pot.NewDialog(textWidget, "Edit Comment")
	dialog.SetLabels("Save (Alt+Enter: newline)", "Cancel")

	return CommentEditor{
		dialog:   dialog,
		textarea: ta,
		active:   false,
	}
}

// textareaWidget wraps the bubbletea textarea as a teapot Widget
type textareaWidget struct {
	pot.BaseWidget
	textarea textarea.Model
}

func (t *textareaWidget) Render(buf *pot.SubBuffer) {
	view := t.textarea.View()
	lines := strings.Split(view, "\n")
	for y, line := range lines {
		if y >= buf.Height() {
			break
		}
		x := 0
		for _, r := range line {
			if x >= buf.Width() {
				break
			}
			buf.SetCell(x, y, pot.Cell{Rune: r})
			x++
		}
	}
}

func (t *textareaWidget) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Don't handle Enter or Escape - let the dialog handle those
	if msg.Type == tea.KeyEnter || msg.Type == tea.KeyEsc {
		return false, nil
	}
	var cmd tea.Cmd
	t.textarea, cmd = t.textarea.Update(msg)
	return true, cmd
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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Alt+Enter inserts a newline
			if msg.Alt {
				m.textarea.InsertString("\n")
				// Update the widget's textarea too
				if tw, ok := m.dialog.Content().(*textareaWidget); ok {
					tw.textarea = m.textarea
				}
				return nil
			}
			// Plain Enter saves and exits
			return m.saveComment()

		case tea.KeyEsc:
			// Cancel - just close without saving
			m.Deactivate()
			return nil

		default:
			// Pass other keys to textarea
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			// Update the widget's textarea too
			if tw, ok := m.dialog.Content().(*textareaWidget); ok {
				tw.textarea = m.textarea
			}
			return cmd
		}
	}

	return nil
}

// View renders the comment editor
func (m CommentEditor) View() string {
	if !m.active {
		return ""
	}

	// Sync textarea state to widget
	if tw, ok := m.dialog.Content().(*textareaWidget); ok {
		tw.textarea = m.textarea
	}

	// Create a buffer and render the dialog
	buf := pot.NewBuffer(m.width, m.height)
	sub := buf.Sub(buf.Bounds())
	m.dialog.Render(sub)

	return buf.String()
}

// Activate activates the comment editor for a specific line
func (m *CommentEditor) Activate(lineNum int, existingComment string) tea.Cmd {
	m.active = true
	m.lineNum = lineNum
	m.textarea.SetValue(existingComment)
	m.textarea.Focus()
	// Sync to widget
	if tw, ok := m.dialog.Content().(*textareaWidget); ok {
		tw.textarea = m.textarea
	}
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
	// Reserve space for title and buttons
	m.textarea.SetWidth(width - 4)
	m.textarea.SetHeight(height - 4)
	m.dialog.SetBounds(pot.NewRect(0, 0, width, height))
}

// CommentSavedMsg is sent when a comment is saved
type CommentSavedMsg struct {
	LineNum int
	Comment string
	Exit    bool
}

// saveComment saves the current comment
func (m *CommentEditor) saveComment() tea.Cmd {
	comment := m.GetComment()
	lineNum := m.lineNum

	m.Deactivate()

	return func() tea.Msg {
		return CommentSavedMsg{
			LineNum: lineNum,
			Comment: comment,
			Exit:    true,
		}
	}
}
