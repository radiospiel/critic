package ui

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/pkg/critic"
	pot "git.15b.it/eno/critic/teapot"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommentEditor represents the comment editing UI using a teapot Dialog.
// It shows the conversation history at the top and a textarea for replies at the bottom.
type CommentEditor struct {
	dialog       *pot.Dialog
	historyView  *conversationHistoryWidget
	textarea     textarea.Model
	active       bool
	lineNum      int
	width        int
	height       int
	conversation *critic.Conversation
	isNewComment bool // true if creating a new comment (no history to show)
}

// NewCommentEditor creates a new comment editor
func NewCommentEditor() CommentEditor {
	ta := textarea.New()
	ta.Placeholder = "Enter your reply..."
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

	// Create the history view widget
	historyView := newConversationHistoryWidget()

	// Create a combined widget with history + textarea
	contentWidget := &commentEditorContent{
		historyView: historyView,
		textarea:    ta,
	}

	dialog := pot.NewDialog(contentWidget, "Reply to Comment")
	dialog.SetLabels("Save", "Cancel")
	dialog.SetBorderFooter("Ctrl+S: Save │ Esc: Cancel")

	return CommentEditor{
		dialog:      dialog,
		historyView: historyView,
		textarea:    ta,
		active:      false,
	}
}

// conversationHistoryWidget displays the conversation history in a scrollable view
type conversationHistoryWidget struct {
	pot.BaseWidget
	lines        []string
	scrollOffset int
}

func newConversationHistoryWidget() *conversationHistoryWidget {
	return &conversationHistoryWidget{}
}

func (h *conversationHistoryWidget) SetConversation(conv *critic.Conversation) {
	h.lines = nil
	h.scrollOffset = 0

	if conv == nil || len(conv.Messages) == 0 {
		return
	}

	// Build lines from conversation messages
	for _, msg := range conv.Messages {
		// Use emojis for author
		prefix := "👤" // Human
		if msg.Author == critic.AuthorAI {
			prefix = "🤖" // AI/Robot
		}

		// Format timestamp as HH:MM:SS
		timestamp := msg.CreatedAt.Format("15:04:05")

		// Split message into lines
		msgLines := strings.Split(msg.Message, "\n")
		for i, line := range msgLines {
			if i == 0 {
				h.lines = append(h.lines, fmt.Sprintf("%s [%s] %s", prefix, timestamp, line))
			} else {
				// Indent continuation lines
				h.lines = append(h.lines, "              "+line)
			}
		}
		// Add blank line between messages
		h.lines = append(h.lines, "")
	}

	// Remove trailing blank line
	if len(h.lines) > 0 && h.lines[len(h.lines)-1] == "" {
		h.lines = h.lines[:len(h.lines)-1]
	}
}

func (h *conversationHistoryWidget) Render(buf *pot.SubBuffer) {
	width := buf.Width()
	height := buf.Height()

	// Style for history (slightly dimmed)
	historyStyle := lipgloss.NewStyle().Faint(true)

	for y := 0; y < height; y++ {
		lineIdx := h.scrollOffset + y
		var line string
		if lineIdx < len(h.lines) {
			line = h.lines[lineIdx]
		}

		// Truncate or pad to width
		visibleWidth := lipgloss.Width(line)
		if visibleWidth > width {
			line = truncateString(line, width)
		}

		// Render with style
		styled := historyStyle.Render(line)
		cells := pot.ParseANSILine(styled)

		for x := 0; x < width; x++ {
			if x < len(cells) {
				buf.SetCell(x, y, cells[x])
			} else {
				buf.SetCell(x, y, pot.Cell{Rune: ' '})
			}
		}
	}
}

func (h *conversationHistoryWidget) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// History view handles scroll keys
	height := h.Bounds().Height
	maxScroll := max(0, len(h.lines)-height)

	switch msg.String() {
	case "up", "k":
		if h.scrollOffset > 0 {
			h.scrollOffset--
		}
		return true, nil
	case "down", "j":
		if h.scrollOffset < maxScroll {
			h.scrollOffset++
		}
		return true, nil
	case "pgup":
		h.scrollOffset = max(0, h.scrollOffset-height)
		return true, nil
	case "pgdown":
		h.scrollOffset = min(maxScroll, h.scrollOffset+height)
		return true, nil
	case "home", "g":
		h.scrollOffset = 0
		return true, nil
	case "end", "G":
		h.scrollOffset = maxScroll
		return true, nil
	}
	return false, nil
}

func (h *conversationHistoryWidget) LineCount() int {
	return len(h.lines)
}

// truncateString truncates a string to the given width
func truncateString(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

// commentEditorContent combines history view and textarea
type commentEditorContent struct {
	pot.BaseWidget
	historyView  *conversationHistoryWidget
	textarea     textarea.Model
	focusOnInput bool // true when focus is on textarea
	showHistory  bool // true when there's history to show
}

func (c *commentEditorContent) SetBounds(bounds pot.Rect) {
	c.BaseWidget.SetBounds(bounds)

	if !c.showHistory {
		// No history - textarea takes full height
		c.textarea.SetWidth(bounds.Width)
		c.textarea.SetHeight(bounds.Height)
		return
	}

	// Split: history gets top portion, textarea gets bottom
	historyLines := c.historyView.LineCount()
	maxHistoryHeight := bounds.Height * 2 / 3 // Max 2/3 for history
	minTextareaHeight := 3                     // Minimum textarea height

	historyHeight := min(historyLines+1, maxHistoryHeight) // +1 for separator
	textareaHeight := bounds.Height - historyHeight - 1    // -1 for separator

	if textareaHeight < minTextareaHeight {
		textareaHeight = minTextareaHeight
		historyHeight = bounds.Height - textareaHeight - 1
	}

	c.historyView.SetBounds(pot.NewRect(bounds.X, bounds.Y, bounds.Width, historyHeight))
	c.textarea.SetWidth(bounds.Width)
	c.textarea.SetHeight(textareaHeight)
}

func (c *commentEditorContent) Render(buf *pot.SubBuffer) {
	width := buf.Width()
	height := buf.Height()

	if !c.showHistory {
		// Just render textarea
		c.renderTextarea(buf, 0, height)
		return
	}

	// Calculate layout
	historyLines := c.historyView.LineCount()
	maxHistoryHeight := height * 2 / 3
	minTextareaHeight := 3

	historyHeight := min(historyLines+1, maxHistoryHeight)
	textareaHeight := height - historyHeight - 1

	if textareaHeight < minTextareaHeight {
		textareaHeight = minTextareaHeight
		historyHeight = height - textareaHeight - 1
	}

	// Render history directly into buffer
	c.renderHistory(buf, 0, historyHeight, width)

	// Render separator
	separatorY := historyHeight
	separatorStyle := lipgloss.NewStyle().Faint(true)
	separator := strings.Repeat("─", width)
	styled := separatorStyle.Render(separator)
	cells := pot.ParseANSILine(styled)
	for x := 0; x < width; x++ {
		if x < len(cells) {
			buf.SetCell(x, separatorY, cells[x])
		} else {
			buf.SetCell(x, separatorY, pot.Cell{Rune: '─'})
		}
	}

	// Render textarea
	textareaY := separatorY + 1
	c.renderTextarea(buf, textareaY, textareaHeight)
}

func (c *commentEditorContent) renderHistory(buf *pot.SubBuffer, startY, height, width int) {
	// Style for history (slightly dimmed)
	historyStyle := lipgloss.NewStyle().Faint(true)

	for y := 0; y < height; y++ {
		bufY := startY + y
		if bufY >= buf.Height() {
			break
		}

		lineIdx := c.historyView.scrollOffset + y
		var line string
		if lineIdx < len(c.historyView.lines) {
			line = c.historyView.lines[lineIdx]
		}

		// Truncate if needed
		visibleWidth := lipgloss.Width(line)
		if visibleWidth > width {
			line = truncateString(line, width)
		}

		// Render with style
		styled := historyStyle.Render(line)
		cells := pot.ParseANSILine(styled)

		for x := 0; x < width; x++ {
			if x < len(cells) {
				buf.SetCell(x, bufY, cells[x])
			} else {
				buf.SetCell(x, bufY, pot.Cell{Rune: ' '})
			}
		}
	}
}

func (c *commentEditorContent) renderTextarea(buf *pot.SubBuffer, startY, height int) {
	view := c.textarea.View()
	lines := strings.Split(view, "\n")

	width := buf.Width()

	for y := 0; y < height; y++ {
		bufY := startY + y
		if bufY >= buf.Height() {
			break
		}

		var line string
		if y < len(lines) {
			line = lines[y]
		}

		cells := pot.ParseANSILine(line)

		for x := 0; x < width; x++ {
			if x < len(cells) {
				buf.SetCell(x, bufY, cells[x])
			} else {
				buf.SetCell(x, bufY, pot.Cell{Rune: ' '})
			}
		}
	}
}

func (c *commentEditorContent) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Don't handle Escape or Ctrl+S - let the dialog/editor handle those
	if msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlS {
		return false, nil
	}

	// Tab switches focus between history and textarea
	if msg.Type == tea.KeyTab && c.showHistory {
		c.focusOnInput = !c.focusOnInput
		return true, nil
	}

	// If focus is on history, let it handle scroll keys
	if !c.focusOnInput && c.showHistory {
		if handled, cmd := c.historyView.HandleKey(msg); handled {
			return handled, cmd
		}
	}

	// Otherwise, pass to textarea
	var cmd tea.Cmd
	c.textarea, cmd = c.textarea.Update(msg)
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
		case tea.KeyCtrlS:
			// Ctrl+S saves and exits
			return m.saveComment()

		case tea.KeyEsc:
			// Cancel - just close without saving
			m.Deactivate()
			return nil

		default:
			// Pass other keys to content widget
			if content, ok := m.dialog.Content().(*commentEditorContent); ok {
				handled, cmd := content.HandleKey(msg)
				if handled {
					// Sync textarea state
					m.textarea = content.textarea
				}
				return cmd
			}
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
	if content, ok := m.dialog.Content().(*commentEditorContent); ok {
		content.textarea = m.textarea
	}

	// Create a buffer and render the dialog using RenderWidget for proper border handling
	buf := pot.NewBuffer(m.width, m.height)
	sub := buf.Sub(buf.Bounds())
	pot.RenderWidget(m.dialog, sub)

	return buf.String()
}

// ActivateWithConversation activates the comment editor with a full conversation
func (m *CommentEditor) ActivateWithConversation(lineNum int, conv *critic.Conversation) tea.Cmd {
	m.active = true
	m.lineNum = lineNum
	m.conversation = conv
	m.isNewComment = conv == nil || len(conv.Messages) == 0
	m.textarea.SetValue("")
	m.textarea.Focus()

	// Update dialog title
	if m.isNewComment {
		m.dialog.SetTitle("New Comment")
		m.textarea.Placeholder = "Enter your comment..."
	} else {
		m.dialog.SetTitle(fmt.Sprintf("Reply to Comment (line %d)", lineNum))
		m.textarea.Placeholder = "Enter your reply..."
	}

	// Update content widget
	if content, ok := m.dialog.Content().(*commentEditorContent); ok {
		content.historyView.SetConversation(conv)
		content.showHistory = !m.isNewComment
		content.focusOnInput = true
		content.textarea = m.textarea
	}

	return textarea.Blink
}

// Activate activates the comment editor for a specific line (legacy, creates new comment)
func (m *CommentEditor) Activate(lineNum int, existingComment string) tea.Cmd {
	// Legacy behavior - treat as new comment or editing existing
	m.active = true
	m.lineNum = lineNum
	m.conversation = nil
	m.isNewComment = true
	m.textarea.SetValue(existingComment)
	m.textarea.Focus()

	m.dialog.SetTitle("Edit Comment")
	m.textarea.Placeholder = "Enter your comment..."

	if content, ok := m.dialog.Content().(*commentEditorContent); ok {
		content.historyView.SetConversation(nil)
		content.showHistory = false
		content.focusOnInput = true
		content.textarea = m.textarea
	}

	return textarea.Blink
}

// Deactivate deactivates the comment editor
func (m *CommentEditor) Deactivate() {
	m.active = false
	m.textarea.Blur()
	m.textarea.SetValue("")
	m.conversation = nil
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

// GetConversationUUID returns the UUID of the conversation being replied to
func (m CommentEditor) GetConversationUUID() string {
	if m.conversation != nil {
		return m.conversation.UUID
	}
	return ""
}

// IsReply returns true if this is a reply to an existing conversation
func (m CommentEditor) IsReply() bool {
	return m.conversation != nil && len(m.conversation.Messages) > 0
}

// SetSize sets the size of the comment editor
func (m *CommentEditor) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Reserve space for border
	innerWidth := width - 4
	innerHeight := height - 4
	m.textarea.SetWidth(innerWidth)
	m.textarea.SetHeight(innerHeight)
	m.dialog.SetBounds(pot.NewRect(0, 0, width, height))

	// Update content widget bounds
	if content, ok := m.dialog.Content().(*commentEditorContent); ok {
		content.SetBounds(pot.NewRect(0, 0, innerWidth, innerHeight))
	}
}

// CommentSavedMsg is sent when a comment is saved
type CommentSavedMsg struct {
	LineNum          int
	Comment          string
	Exit             bool
	ConversationUUID string // UUID of conversation if this is a reply
	IsReply          bool   // true if this is a reply to an existing conversation
}

// saveComment saves the current comment
func (m *CommentEditor) saveComment() tea.Cmd {
	comment := m.GetComment()
	lineNum := m.lineNum
	convUUID := m.GetConversationUUID()
	isReply := m.IsReply()

	m.Deactivate()

	return func() tea.Msg {
		return CommentSavedMsg{
			LineNum:          lineNum,
			Comment:          comment,
			Exit:             true,
			ConversationUUID: convUUID,
			IsReply:          isReply,
		}
	}
}
