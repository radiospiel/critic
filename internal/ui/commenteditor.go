package ui

import (
	"strings"

	pot "git.15b.it/eno/critic/teapot"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	// Configure plain styles - remove cursor line highlighting and prompt indicator
	plainStyle := lipgloss.NewStyle()
	ta.FocusedStyle = textarea.Style{
		Base:             plainStyle,
		CursorLine:       plainStyle, // No highlight on cursor line
		CursorLineNumber: plainStyle,
		EndOfBuffer:      plainStyle,
		LineNumber:       plainStyle,
		Placeholder:      plainStyle.Faint(true),
		Prompt:           plainStyle, // No left indicator
		Text:             plainStyle,
	}
	ta.BlurredStyle = ta.FocusedStyle

	// Create a wrapper widget for the textarea
	textWidget := &textareaWidget{textarea: ta}

	dialog := pot.NewDialog(textWidget, "Edit Comment")
	dialog.SetLabels("Save", "Cancel")
	dialog.SetBorderFooter("Enter: Save │ Esc: Cancel │ Alt+Enter: Newline")

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

	width := buf.Width()
	height := buf.Height()

	for y := 0; y < height; y++ {
		// Get line content or empty if past end
		var line string
		if y < len(lines) {
			line = lines[y]
		}

		// Parse ANSI-encoded line to cells with styles
		cells := pot.ParseANSILine(line)

		// Render cells, filling to width
		for x := 0; x < width; x++ {
			if x < len(cells) {
				buf.SetCell(x, y, cells[x])
			} else {
				buf.SetCell(x, y, pot.Cell{Rune: ' '})
			}
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

	// Create a buffer and render the dialog using RenderWidget for proper border handling
	buf := pot.NewBuffer(m.width, m.height)
	sub := buf.Sub(buf.Bounds())
	pot.RenderWidget(m.dialog, sub)

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
