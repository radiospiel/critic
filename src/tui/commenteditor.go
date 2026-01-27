package tui

import (
	"fmt"
	"strings"

	"github.com/radiospiel/critic/src/pkg/critic"
	pot "github.com/radiospiel/critic/teapot"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommentEditor represents the comment editing UI using a teapot ModalDialog.
// It shows the conversation history at the top and a textarea for replies at the bottom.
// CommentEditor embeds ModalDialog to inherit its Close() method for proper modal cleanup.
type CommentEditor struct {
	*pot.ModalDialog // embedded - inherits Close() and other dialog methods
	historyView      *conversationHistoryView
	textareaView     *pot.TextAreaView
	active           bool
	lineNum          int
	conversation     *critic.Conversation
}

// NewCommentEditor creates a new comment editor
func NewCommentEditor() CommentEditor {
	ta := textarea.New()
	ta.Placeholder = "Enter your reply..."
	// TODO(bot): add this to config
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

	// Create the views
	historyView := newConversationHistoryWidget()
	textareaView := pot.NewTextAreaView(ta)
	separator := pot.NewSeparatorView()
	vbox := pot.NewVBox(0)

	// Create a combined widget with history + textarea using VBox layout
	contentWidget := &commentEditorContent{
		vbox:         vbox,
		historyView:  historyView,
		separator:    separator,
		textareaView: textareaView,
	}

	dialog := pot.NewModalDialog(contentWidget, "Reply to Comment")
	dialog.SetLabels("Save", "Cancel")
	dialog.SetBorderFooter("Ctrl+S: Save │ Esc: Cancel")

	return CommentEditor{
		ModalDialog:  dialog,
		historyView:  historyView,
		textareaView: textareaView,
		active:       false,
	}
}

// conversationHistoryView displays the conversation history in a scrollable view
type conversationHistoryView struct {
	pot.BaseView
	conversation   *critic.Conversation // Store conversation for re-wrapping
	lines          []string
	lastWidth      int  // Last width used for wrapping
	scrollOffset   int
	initialScrolled bool // Whether initial scroll to end has been done
}

func newConversationHistoryWidget() *conversationHistoryView {
	return &conversationHistoryView{}
}

func (h *conversationHistoryView) SetConversation(conv *critic.Conversation) {
	h.conversation = conv
	h.lines = nil
	h.lastWidth = 0
	h.scrollOffset = 0
	h.initialScrolled = false
}

// rebuildLines rebuilds the wrapped lines for the given width
func (h *conversationHistoryView) rebuildLines(width int) {
	h.lines = nil
	h.lastWidth = width
	h.scrollOffset = 0 // Reset scroll, will be set to end after building

	conv := h.conversation
	if conv == nil || len(conv.Messages) == 0 {
		return
	}

	// Message content is indented by 4 spaces
	const indent = "    "
	wrapWidth := max(width-len(indent), 20)

	// Build lines from conversation messages
	for _, msg := range conv.Messages {
		// Use text labels for author
		prefix := "You" // Human
		if msg.Author == critic.AuthorAI {
			prefix = "AI"
		}

		// Format timestamp as HH:MM:SS
		timestamp := msg.CreatedAt.Format("15:04:05")

		// Header on its own line
		h.lines = append(h.lines, fmt.Sprintf("%s [%s]", prefix, timestamp))

		// Parse message with markdown-aware line wrapping
		wrappedLines := wrapMessageWithMarkdown(msg.Message, wrapWidth)
		for _, line := range wrappedLines {
			// All content lines are indented
			h.lines = append(h.lines, indent+line)
		}
		// Add blank line between messages
		h.lines = append(h.lines, "")
	}

	// Remove trailing blank line
	if len(h.lines) > 0 && h.lines[len(h.lines)-1] == "" {
		h.lines = h.lines[:len(h.lines)-1]
	}
}

// wrapMessageWithMarkdown wraps text respecting markdown code blocks.
// Regular text is wrapped at word boundaries, code blocks are preserved as-is.
func wrapMessageWithMarkdown(text string, width int) []string {
	if width <= 0 {
		width = 80
	}

	var result []string
	lines := strings.Split(text, "\n")
	inCodeBlock := false

	for _, line := range lines {
		// Check for code block markers
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			result = append(result, line) // Keep the marker line as-is
			continue
		}

		if inCodeBlock {
			// Don't wrap code blocks
			result = append(result, line)
		} else {
			// Wrap regular text
			wrapped := wrapTextLine(line, width)
			result = append(result, wrapped...)
		}
	}

	return result
}

// wrapTextLine wraps a single line of text at word boundaries
func wrapTextLine(line string, width int) []string {
	if len(line) == 0 {
		return []string{""}
	}

	// If line fits, return as-is
	if len(line) <= width {
		return []string{line}
	}

	var result []string
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{line}
	}

	// Preserve leading whitespace
	leadingSpace := ""
	for _, r := range line {
		if r == ' ' || r == '\t' {
			leadingSpace += string(r)
		} else {
			break
		}
	}

	current := leadingSpace
	for _, word := range words {
		if len(current) == 0 || current == leadingSpace {
			current = leadingSpace + word
		} else if len(current)+1+len(word) <= width {
			current += " " + word
		} else {
			result = append(result, current)
			current = leadingSpace + word
		}
	}
	if len(current) > 0 {
		result = append(result, current)
	}

	return result
}

func (h *conversationHistoryView) Render(buf *pot.SubBuffer) {
	width := buf.Width()
	height := buf.Height()

	// Rebuild lines if width changed
	if width != h.lastWidth {
		h.rebuildLines(width)
	}

	totalLines := len(h.lines)

	// Calculate max scroll - when scrolled, hint takes 1 line
	maxScroll := 0
	if totalLines > height {
		maxScroll = totalLines - height + 1
	}

	// Auto-scroll to end on first render (show most recent content)
	if !h.initialScrolled && totalLines > height {
		h.scrollOffset = maxScroll
		h.initialScrolled = true
	}

	// Clamp scroll offset to valid range
	if h.scrollOffset < 0 {
		h.scrollOffset = 0
	}
	if h.scrollOffset > maxScroll {
		h.scrollOffset = maxScroll
	}

	// Style for history (slightly dimmed)
	historyStyle := lipgloss.NewStyle().Faint(true)
	hintStyle := lipgloss.NewStyle().Faint(true).Italic(true)

	// Check if there are hidden lines above
	hiddenAbove := h.scrollOffset
	showHint := hiddenAbove > 0

	startY := 0
	renderHeight := height
	if showHint {
		// Reserve first line for hint
		startY = 1
		renderHeight = height - 1

		// Render scroll hint
		hint := fmt.Sprintf("↑ %d more lines above (use ↑/↓ to scroll)", hiddenAbove)
		if len(hint) > width {
			hint = hint[:width]
		}
		styled := hintStyle.Render(hint)
		parsedCells := pot.ParseANSILine(styled)
		rowCells := make([]pot.Cell, width)
		for x := range width {
			if x < len(parsedCells) {
				rowCells[x] = parsedCells[x]
			} else {
				rowCells[x] = pot.Cell{Rune: ' '}
			}
		}
		buf.SetCells(0, 0, rowCells)
	}

	// Render history lines
	for y := range renderHeight {
		bufY := startY + y
		if bufY >= buf.Height() {
			break
		}

		lineIdx := h.scrollOffset + y
		var line string
		if lineIdx < len(h.lines) {
			line = h.lines[lineIdx]
		}

		// Truncate if needed
		visibleWidth := lipgloss.Width(line)
		if visibleWidth > width {
			line = truncateString(line, width)
		}

		// Render with style
		styled := historyStyle.Render(line)
		parsedCells := pot.ParseANSILine(styled)

		// Build row with padding
		rowCells := make([]pot.Cell, width)
		for x := range width {
			if x < len(parsedCells) {
				rowCells[x] = parsedCells[x]
			} else {
				rowCells[x] = pot.Cell{Rune: ' '}
			}
		}
		buf.SetCells(0, bufY, rowCells)
	}
}

func (h *conversationHistoryView) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// History view handles scroll keys
	height := h.Bounds().Height
	totalLines := len(h.lines)

	// Calculate max scroll - can't scroll past the point where last line is visible
	// When scrollOffset > 0, the scroll hint takes 1 line, so only (height-1) content lines visible
	// When scrollOffset == 0, all height lines are available for content
	maxScroll := 0
	if totalLines > height {
		// When scrolled, hint takes 1 line, so we need: scrollOffset + (height-1) >= totalLines
		// Thus: maxScroll = totalLines - height + 1
		maxScroll = totalLines - height + 1
	}

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

func (h *conversationHistoryView) LineCount() int {
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

// commentEditorContent combines history view and textarea using a VBox layout.
type commentEditorContent struct {
	pot.BaseView
	vbox         *pot.BoxLayout
	historyView  *conversationHistoryView
	separator    *pot.SeparatorView
	textareaView *pot.TextAreaView
	focusOnInput bool // true when focus is on textarea
	showHistory  bool // true when there's history to show
}

// calcTextareaHeight returns the height needed for the textarea based on its content.
// It counts the number of lines in the textarea and returns at least minHeight.
// When maxHeight > 0, the height is capped at that value.
func (c *commentEditorContent) calcTextareaHeight(minHeight, maxHeight int) int {
	content := c.textareaView.Value()
	if content == "" {
		return minHeight
	}
	lines := strings.Count(content, "\n") + 1
	if lines < minHeight {
		return minHeight
	}
	if maxHeight > 0 && lines > maxHeight {
		return maxHeight
	}
	return lines
}

func (c *commentEditorContent) SetBounds(bounds pot.Rect) {
	c.BaseView.SetBounds(bounds)
	c.updateLayout()
	c.vbox.SetBounds(bounds)
}

// updateLayout updates the VBox children and constraints based on current state.
func (c *commentEditorContent) updateLayout() {
	bounds := c.Bounds()
	c.vbox.ClearChildren()

	if !c.showHistory {
		// No history - textarea takes full height
		c.textareaView.SetConstraints(pot.Constraints{
			MinHeight:       bounds.Height,
			VerticalStretch: 1,
		})
		c.vbox.AddChild(c.textareaView)
		return
	}

	// Ensure lines are built before calculating layout
	if bounds.Width != c.historyView.lastWidth {
		c.historyView.rebuildLines(bounds.Width)
	}

	// Calculate textarea height based on content (minimum 3, max half the space)
	maxTextareaHeight := bounds.Height / 2
	textareaHeight := c.calcTextareaHeight(3, maxTextareaHeight)

	// History view: takes remaining space (with stretch)
	c.historyView.SetConstraints(pot.Constraints{
		MinHeight:       1,
		VerticalStretch: 1,
	})

	// Textarea: fixed preferred height based on content
	c.textareaView.SetConstraints(pot.Constraints{
		MinHeight:       3,
		PreferredHeight: textareaHeight,
	})

	c.vbox.AddChild(c.historyView)
	c.vbox.AddChild(c.separator)
	c.vbox.AddChild(c.textareaView)
}

func (c *commentEditorContent) Render(buf *pot.SubBuffer) {
	// Delegate rendering to VBox
	c.vbox.Render(buf)
}

// Children returns the VBox's children for focus traversal.
func (c *commentEditorContent) Children() []pot.View {
	return c.vbox.Children()
}

func (c *commentEditorContent) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Don't handle Escape or Ctrl+S - let the dialog/editor handle those
	if msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlS {
		return false, nil
	}

	// Up/down/pgup/pgdown scroll the history (if there's history to show)
	if c.showHistory {
		switch msg.String() {
		case "up", "pgup":
			if handled, cmd := c.historyView.HandleKey(msg); handled {
				return handled, cmd
			}
		case "down", "pgdown":
			if handled, cmd := c.historyView.HandleKey(msg); handled {
				return handled, cmd
			}
		}
	}

	// Other keys go to textarea view
	return c.textareaView.HandleKey(msg)
}

// Init initializes the comment editor
func (m CommentEditor) Init() tea.Cmd {
	return nil
}

// HandleKey implements teapot.ModalKeyHandler for routing keys via the focus manager.
// When the comment editor is active as a modal, all keys are routed through this method.
func (m *CommentEditor) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	if !m.active {
		return false, nil
	}

	switch msg.Type {
	case tea.KeyCtrlS:
		// Ctrl+S saves and exits
		return true, m.saveComment()

	case tea.KeyEsc:
		// Cancel - just close without saving
		m.Deactivate()
		return true, nil

	default:
		// Pass other keys to content widget
		if content, ok := m.ModalDialog.Content().(*commentEditorContent); ok {
			_, cmd := content.HandleKey(msg)
			return true, cmd // Modal captures all keys when active
		}
	}

	return true, nil // Modal captures all keys
}

// Update handles messages for the comment editor
func (m *CommentEditor) Update(msg tea.Msg) tea.Cmd {
	if !m.active {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		_, cmd := m.HandleKey(msg)
		return cmd
	}

	return nil
}

// Render renders the comment editor to a buffer
func (m *CommentEditor) Render(buf *pot.SubBuffer) {
	if !m.active {
		return
	}

	// Render the dialog using RenderWidget for proper border handling
	pot.RenderWidget(m.ModalDialog, buf)
}

// RenderOverlay renders the comment editor as an overlay on top of the base view
// with the background dimmed. This is the standard way to display the editor.
func (m *CommentEditor) RenderOverlay(baseView string, screenWidth, screenHeight int) string {
	if !m.active {
		return baseView
	}

	// Use the ModalDialog's RenderOverlay for dimming and centering
	return m.ModalDialog.RenderOverlay(baseView, screenWidth, screenHeight)
}

// ActivateWithConversation activates the comment editor with a full conversation
func (m *CommentEditor) ActivateWithConversation(lineNum int, conv *critic.Conversation, debug bool) tea.Cmd {
	m.active = true
	m.lineNum = lineNum
	m.conversation = conv
	m.textareaView.SetValue("")
	m.textareaView.Focus()

	isNewComment := conv == nil || len(conv.Messages) == 0

	// Update dialog title and placeholder via underlying model
	ta := m.textareaView.Model()
	if isNewComment {
		m.ModalDialog.SetTitle("New Comment")
		ta.Placeholder = "Enter your comment..."
	} else {
		title := fmt.Sprintf("Reply to Comment (line %d)", lineNum)
		if debug && conv.UUID != "" {
			title = fmt.Sprintf("Reply to Comment (line %d) [%s]", lineNum, conv.UUID)
		}
		m.ModalDialog.SetTitle(title)
		ta.Placeholder = "Enter your reply..."
	}
	m.textareaView.SetModel(ta)

	// Update content widget
	if content, ok := m.ModalDialog.Content().(*commentEditorContent); ok {
		content.historyView.SetConversation(conv)
		content.showHistory = !isNewComment
		content.focusOnInput = true
		// Trigger layout update with current bounds
		content.SetBounds(content.Bounds())
	}

	return textarea.Blink
}

// Deactivate deactivates the comment editor and clears it from the focus manager's modal.
func (m *CommentEditor) Deactivate() {
	m.active = false
	m.textareaView.Blur()
	m.textareaView.SetValue("")
	m.conversation = nil
	m.Close() // Clear modal from focus manager (inherited from ModalDialog)
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
	return strings.TrimSpace(m.textareaView.Value())
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
	m.ModalDialog.SetBounds(pot.NewRect(0, 0, width, height))

	// Update content widget bounds (inside the border)
	innerWidth := width - 4
	innerHeight := height - 4
	if content, ok := m.ModalDialog.Content().(*commentEditorContent); ok {
		content.SetBounds(pot.NewRect(0, 0, innerWidth, innerHeight))
	}
}

// Width returns the width of the comment editor
func (m *CommentEditor) Width() int {
	return m.Bounds().Width
}

// Height returns the height of the comment editor
func (m *CommentEditor) Height() int {
	return m.Bounds().Height
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
