package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.15b.it/eno/critic/internal/config"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/highlight"
	"git.15b.it/eno/critic/internal/logger"
	"git.15b.it/eno/critic/pkg/critic"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiffViewModel represents the diff viewer pane
type DiffViewModel struct {
	file                *ctypes.FileDiff
	viewport            viewport.Model
	width               int
	height              int
	ready               bool
	highlighter         *highlight.Highlighter
	cachedContent       string
	cachedFile          *ctypes.FileDiff
	highlightingEnabled bool
	highlightTime       time.Duration // Accumulated syntax highlighting time
	cursorLine          int           // Current active line (0-based)
	totalLines          int           // Total number of lines in rendered diff
	focused              bool                // Whether this pane is focused
	navigableLines       []int               // Line numbers that can have cursor (diff lines only)
	messaging            critic.Messaging
	commentLines         map[int]int         // Maps rendered line number to source line number for comment lines
	sourceLines          map[int]int         // Maps rendered line number to source line number for all diff lines
	preserveCursorLine   int                 // Source line to restore cursor to after refresh (0 = don't preserve)
	lineConversationUUID map[int]string      // Maps rendered line number to conversation UUID
	gotoBottomOnLoad     bool                // If true, go to bottom after next file load
}

// NewDiffViewModel creates a new diff viewer model
func NewDiffViewModel() DiffViewModel {
	return DiffViewModel{
		highlighter:         highlight.NewHighlighter(),
		highlightingEnabled: true, // Default to enabled
	}
}

// Init initializes the diff view model
func (m DiffViewModel) Init() tea.Cmd {
	return nil
}

// Update updates the diff view model
func (m *DiffViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.ready {
			switch msg.String() {
			case "up", "k":
				if m.moveCursorUp() {
					m.ensureCursorVisible()
					return m.refreshContent()
				} else if m.isAtTop() {
					// At top of file, request previous file
					return func() tea.Msg { return RequestPrevFileMsg{} }
				}
			case "down", "j":
				if m.moveCursorDown() {
					m.ensureCursorVisible()
					return m.refreshContent()
				} else if m.isAtBottom() {
					// At bottom of file, request next file
					return func() tea.Msg { return RequestNextFileMsg{} }
				}
			case "shift+up":
				if m.moveCursorUpN(config.ShiftArrowJumpSize) {
					m.ensureCursorVisible()
					return m.refreshContent()
				} else if m.isAtTop() {
					return func() tea.Msg { return RequestPrevFileMsg{} }
				}
			case "shift+down":
				if m.moveCursorDownN(config.ShiftArrowJumpSize) {
					m.ensureCursorVisible()
					return m.refreshContent()
				} else if m.isAtBottom() {
					return func() tea.Msg { return RequestNextFileMsg{} }
				}
			case "g": // Go to top
				if len(m.navigableLines) > 0 {
					m.cursorLine = m.navigableLines[0]
					m.viewport.GotoTop()
					return m.refreshContent()
				}
			case "G": // Go to bottom
				if len(m.navigableLines) > 0 {
					m.cursorLine = m.navigableLines[len(m.navigableLines)-1]
					m.viewport.GotoBottom()
					return m.refreshContent()
				}
			case "r": // Resolve comment
				return m.resolveCommentAtCursor()
			default:
				// Let viewport handle other keys (page up/down, etc)
				m.viewport, cmd = m.viewport.Update(msg)
			}
		}

	case diffRenderedMsg:
		// Only apply if still viewing the same file
		if msg.file == m.file {
			m.cachedContent = msg.content
			m.totalLines = msg.totalLines
			m.navigableLines = msg.navigableLines
			if m.ready {
				m.viewport.SetContent(m.cachedContent)

				// Restore cursor position if we're preserving it
				if m.preserveCursorLine > 0 {
					// Find the rendered line that corresponds to the source line
					restored := false
					for renderedLine, sourceLine := range m.sourceLines {
						if sourceLine == m.preserveCursorLine {
							m.cursorLine = renderedLine
							// Ensure cursor is visible
							m.ensureCursorVisible()
							restored = true
							break
						}
					}
					// If we couldn't restore, try to find first comment line for that source
					if !restored {
						for renderedLine, sourceLine := range m.commentLines {
							if sourceLine == m.preserveCursorLine {
								m.cursorLine = renderedLine
								m.ensureCursorVisible()
								restored = true
								break
							}
						}
					}
					// Clear the preserve flag
					m.preserveCursorLine = 0

					// If we couldn't find the line, just go to top
					if !restored {
						m.viewport.GotoTop()
						if len(m.navigableLines) > 0 {
							m.cursorLine = m.navigableLines[0]
						}
					}
				} else if m.gotoBottomOnLoad {
					// Go to bottom (when navigating from previous file)
					m.gotoBottomOnLoad = false
					m.viewport.GotoBottom()
					if len(m.navigableLines) > 0 {
						m.cursorLine = m.navigableLines[len(m.navigableLines)-1]
					}
				} else {
					// Normal behavior: go to top and reset cursor
					m.viewport.GotoTop()
					if len(m.navigableLines) > 0 {
						m.cursorLine = m.navigableLines[0]
					}
				}

				// Re-render to show cursor highlighting at the new position
				m.refreshContent()
			}
		}
	}

	return cmd
}

// View renders the diff view
func (m DiffViewModel) View() string {
	if m.file == nil {
		return lipgloss.NewStyle().
			Padding(1, 2).
			Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#666"}).
			Render("Select a file to view diff")
	}

	if m.file.IsBinary {
		return lipgloss.NewStyle().
			Padding(1, 2).
			Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#666"}).
			Render("Binary file - no diff available")
	}

	if len(m.file.Hunks) == 0 {
		msg := "No changes"
		if m.file.IsNew {
			msg = "New file (empty)"
		} else if m.file.IsDeleted {
			msg = "File deleted"
		}
		return lipgloss.NewStyle().
			Padding(1, 2).
			Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#666"}).
			Render(msg)
	}

	if m.ready {
		return m.viewport.View()
	}

	// Fallback if viewport not ready
	return m.cachedContent
}

// SetFile sets the current file to display
func (m *DiffViewModel) SetFile(file *ctypes.FileDiff) tea.Cmd {
	m.file = file

	// Pre-render and cache the diff content
	if file != nil && (m.cachedFile != file) {
		m.cachedFile = file

		// Return command to render in background if highlighting is enabled
		if m.highlightingEnabled {
			return m.renderDiffAsync(file)
		}

		// Otherwise render immediately (no highlighting)
		content, totalLines, navigableLines := m.renderDiff()
		m.cachedContent = content
		m.totalLines = totalLines
		m.navigableLines = navigableLines
		if m.ready {
			m.viewport.SetContent(m.cachedContent)
			m.viewport.GotoTop()
		}
	}
	return nil
}

// RefreshFile forces a re-render of the current file (used when comments change)
func (m *DiffViewModel) RefreshFile() tea.Cmd {
	if m.file == nil {
		return nil
	}

	// Save current cursor position (as source line) to restore after refresh
	currentSourceLine := m.GetSourceLine(m.cursorLine)
	if currentSourceLine == 0 {
		// Try to get from comment lines
		if sourceLine, ok := m.commentLines[m.cursorLine]; ok {
			currentSourceLine = sourceLine
		}
	}
	m.preserveCursorLine = currentSourceLine

	// Clear cache to force re-render
	m.cachedFile = nil

	// Re-render with highlighting if enabled
	if m.highlightingEnabled {
		return m.renderDiffAsync(m.file)
	}

	// Otherwise render immediately (non-async path)
	content, totalLines, navigableLines := m.renderDiff()
	m.cachedContent = content
	m.totalLines = totalLines
	m.navigableLines = navigableLines
	if m.ready {
		m.viewport.SetContent(m.cachedContent)

		// Restore cursor position for synchronous render
		if m.preserveCursorLine > 0 {
			restored := false
			for renderedLine, sourceLine := range m.sourceLines {
				if sourceLine == m.preserveCursorLine {
					m.cursorLine = renderedLine
					m.ensureCursorVisible()
					restored = true
					break
				}
			}
			if !restored {
				for renderedLine, sourceLine := range m.commentLines {
					if sourceLine == m.preserveCursorLine {
						m.cursorLine = renderedLine
						m.ensureCursorVisible()
						restored = true
						break
					}
				}
			}
			m.preserveCursorLine = 0
		}
	}
	return nil
}

// diffRenderedMsg is sent when async rendering completes
type diffRenderedMsg struct {
	file           *ctypes.FileDiff
	content        string
	totalLines     int
	navigableLines []int
}

// RequestNextFileMsg is sent when the user scrolls past the last line
type RequestNextFileMsg struct{}

// RequestPrevFileMsg is sent when the user scrolls before the first line
type RequestPrevFileMsg struct{}

// renderDiffAsync renders the diff in a background goroutine
func (m *DiffViewModel) renderDiffAsync(file *ctypes.FileDiff) tea.Cmd {
	return func() tea.Msg {
		// Render with highlighting in background
		content, totalLines, navigableLines := m.renderDiff()
		return diffRenderedMsg{
			file:           file,
			content:        content,
			totalLines:     totalLines,
			navigableLines: navigableLines,
		}
	}
}

// refreshContent re-renders the diff with current cursor position
func (m *DiffViewModel) refreshContent() tea.Cmd {
	if m.file == nil {
		return nil
	}
	content, totalLines, navigableLines := m.renderDiff()
	m.cachedContent = content
	m.totalLines = totalLines
	m.navigableLines = navigableLines
	if m.ready {
		m.viewport.SetContent(m.cachedContent)
	}
	return nil
}

// moveCursorUp moves cursor to previous navigable line
func (m *DiffViewModel) moveCursorUp() bool {
	if len(m.navigableLines) == 0 {
		return false
	}
	// Find previous navigable line
	for i := len(m.navigableLines) - 1; i >= 0; i-- {
		if m.navigableLines[i] < m.cursorLine {
			m.cursorLine = m.navigableLines[i]
			return true
		}
	}
	return false
}

// moveCursorDown moves cursor to next navigable line
func (m *DiffViewModel) moveCursorDown() bool {
	if len(m.navigableLines) == 0 {
		return false
	}
	// Find next navigable line
	for i := 0; i < len(m.navigableLines); i++ {
		if m.navigableLines[i] > m.cursorLine {
			m.cursorLine = m.navigableLines[i]
			return true
		}
	}
	return false
}

// moveCursorUpN moves cursor up by n navigable lines
func (m *DiffViewModel) moveCursorUpN(n int) bool {
	if len(m.navigableLines) == 0 {
		return false
	}
	// Find current position in navigable lines
	currentIdx := -1
	for i, line := range m.navigableLines {
		if line == m.cursorLine {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		// Current line not found, find nearest
		for i := len(m.navigableLines) - 1; i >= 0; i-- {
			if m.navigableLines[i] < m.cursorLine {
				currentIdx = i
				break
			}
		}
	}
	if currentIdx <= 0 {
		// Already at top or not found
		if len(m.navigableLines) > 0 && m.cursorLine != m.navigableLines[0] {
			m.cursorLine = m.navigableLines[0]
			return true
		}
		return false
	}
	// Move up by n lines
	newIdx := currentIdx - n
	if newIdx < 0 {
		newIdx = 0
	}
	m.cursorLine = m.navigableLines[newIdx]
	return true
}

// moveCursorDownN moves cursor down by n navigable lines
func (m *DiffViewModel) moveCursorDownN(n int) bool {
	if len(m.navigableLines) == 0 {
		return false
	}
	// Find current position in navigable lines
	currentIdx := -1
	for i, line := range m.navigableLines {
		if line == m.cursorLine {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		// Current line not found, find nearest
		for i := 0; i < len(m.navigableLines); i++ {
			if m.navigableLines[i] > m.cursorLine {
				currentIdx = i - 1
				break
			}
		}
		if currentIdx == -1 {
			currentIdx = len(m.navigableLines) - 1
		}
	}
	if currentIdx >= len(m.navigableLines)-1 {
		// Already at bottom
		return false
	}
	// Move down by n lines
	newIdx := currentIdx + n
	if newIdx >= len(m.navigableLines) {
		newIdx = len(m.navigableLines) - 1
	}
	m.cursorLine = m.navigableLines[newIdx]
	return true
}

// isAtTop returns true if the cursor is at the first navigable line
func (m *DiffViewModel) isAtTop() bool {
	if len(m.navigableLines) == 0 {
		return true
	}
	return m.cursorLine <= m.navigableLines[0]
}

// isAtBottom returns true if the cursor is at the last navigable line
func (m *DiffViewModel) isAtBottom() bool {
	if len(m.navigableLines) == 0 {
		return true
	}
	return m.cursorLine >= m.navigableLines[len(m.navigableLines)-1]
}

// GotoBottom moves the cursor to the last navigable line and scrolls to bottom
func (m *DiffViewModel) GotoBottom() tea.Cmd {
	if len(m.navigableLines) > 0 {
		m.cursorLine = m.navigableLines[len(m.navigableLines)-1]
		if m.ready {
			m.viewport.GotoBottom()
		}
		return m.refreshContent()
	}
	return nil
}

// SetGotoBottomOnLoad sets a flag to go to bottom after the next file load
func (m *DiffViewModel) SetGotoBottomOnLoad() {
	m.gotoBottomOnLoad = true
}

// ensureCursorVisible scrolls the viewport to keep cursor visible
func (m *DiffViewModel) ensureCursorVisible() {
	if !m.ready {
		return
	}
	// Get viewport position
	yOffset := m.viewport.YOffset
	viewHeight := m.viewport.Height

	// If cursor is above viewport, scroll up
	if m.cursorLine < yOffset {
		m.viewport.YOffset = m.cursorLine
	}
	// If cursor is below viewport, scroll down
	if m.cursorLine >= yOffset+viewHeight {
		m.viewport.YOffset = m.cursorLine - viewHeight + 1
	}
}

// ScrollPageDown scrolls down by viewport height minus 3 (but at least 1 row)
// and positions cursor on the second navigable line in the viewport
// If already at the bottom, returns RequestNextFileMsg to load the next file
func (m *DiffViewModel) ScrollPageDown() tea.Cmd {
	if !m.ready || len(m.navigableLines) == 0 {
		return nil
	}

	// Check if we're already at the bottom
	maxOffset := m.totalLines - m.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.viewport.YOffset >= maxOffset && m.isAtBottom() {
		// Already at the bottom, request next file
		return func() tea.Msg { return RequestNextFileMsg{} }
	}

	scrollAmount := m.viewport.Height - 3
	if scrollAmount < 1 {
		scrollAmount = 1
	}

	// Calculate new offset
	newOffset := m.viewport.YOffset + scrollAmount
	if newOffset > maxOffset {
		newOffset = maxOffset
	}

	m.viewport.YOffset = newOffset

	// Position cursor on the second navigable line visible in viewport
	m.positionCursorInViewport(1) // 1 means second line (0-indexed)

	// Refresh to show cursor at new position
	return m.refreshContent()
}

// positionCursorInViewport moves cursor to the nth navigable line in the current viewport
func (m *DiffViewModel) positionCursorInViewport(nth int) {
	if len(m.navigableLines) == 0 {
		return
	}

	viewStart := m.viewport.YOffset
	viewEnd := viewStart + m.viewport.Height

	// Find navigable lines within the current viewport
	visibleNavigableLines := []int{}
	for _, lineNum := range m.navigableLines {
		if lineNum >= viewStart && lineNum < viewEnd {
			visibleNavigableLines = append(visibleNavigableLines, lineNum)
		}
	}

	if len(visibleNavigableLines) == 0 {
		// No navigable lines in viewport, just use the first one after viewStart
		for _, lineNum := range m.navigableLines {
			if lineNum >= viewStart {
				m.cursorLine = lineNum
				return
			}
		}
		// Fallback to last navigable line
		m.cursorLine = m.navigableLines[len(m.navigableLines)-1]
		return
	}

	// Position on the nth visible line (or last if not enough lines)
	if nth >= len(visibleNavigableLines) {
		nth = len(visibleNavigableLines) - 1
	}
	m.cursorLine = visibleNavigableLines[nth]
}

// ScrollPageUp scrolls up by viewport height minus 3 (but at least 1 row)
// and positions cursor on the second navigable line in the viewport
// If already at the top, returns RequestPrevFileMsg to load the previous file
func (m *DiffViewModel) ScrollPageUp() tea.Cmd {
	if !m.ready || len(m.navigableLines) == 0 {
		return nil
	}

	// Check if we're already at the top
	if m.viewport.YOffset <= 0 && m.isAtTop() {
		// Already at the top, request previous file
		return func() tea.Msg { return RequestPrevFileMsg{} }
	}

	scrollAmount := m.viewport.Height - 3
	if scrollAmount < 1 {
		scrollAmount = 1
	}

	// Calculate new offset
	newOffset := m.viewport.YOffset - scrollAmount
	if newOffset < 0 {
		newOffset = 0
	}

	m.viewport.YOffset = newOffset

	// Position cursor on the second navigable line visible in viewport
	m.positionCursorInViewport(1) // 1 means second line (0-indexed)

	// Refresh to show cursor at new position
	return m.refreshContent()
}

// SetFocused sets whether this pane is focused
func (m *DiffViewModel) SetFocused(focused bool) {
	if m.focused != focused {
		m.focused = focused
		// Re-render to update cursor visibility
		m.refreshContent()
	}
}

// GetFile returns the currently displayed file
func (m *DiffViewModel) GetFile() *ctypes.FileDiff {
	return m.file
}

// GetCursorLine returns the current cursor line number
func (m *DiffViewModel) GetCursorLine() int {
	return m.cursorLine
}

// SetSize sets the size of the diff view pane
func (m *DiffViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Initialize viewport if not ready
	if !m.ready {
		m.viewport = viewport.New(width, height)
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = height
	}

	// Experiment: Don't repaint on resize
}

// SetHighlightingEnabled enables or disables syntax highlighting
func (m *DiffViewModel) SetHighlightingEnabled(enabled bool) {
	m.highlightingEnabled = enabled
	// Clear cache to force re-render with new setting
	m.cachedContent = ""
	m.cachedFile = nil
}

// SetMessaging sets the messaging interface for conversations
func (m *DiffViewModel) SetMessaging(messaging critic.Messaging) {
	m.messaging = messaging
}

// GetConversationUUIDAtLine returns the conversation UUID for a rendered line
func (m *DiffViewModel) GetConversationUUIDAtLine(renderedLine int) string {
	if m.lineConversationUUID == nil {
		return ""
	}
	return m.lineConversationUUID[renderedLine]
}

// IsCommentLine returns true if the given line number is a comment line
// and returns the source line number for that comment
func (m *DiffViewModel) IsCommentLine(lineNum int) (bool, int) {
	if m.commentLines == nil {
		return false, 0
	}
	sourceLine, ok := m.commentLines[lineNum]
	return ok, sourceLine
}

// GetSourceLine returns the source line number for a rendered line number
// Returns 0 if the line doesn't map to a source line
func (m *DiffViewModel) GetSourceLine(renderedLine int) int {
	if m.sourceLines == nil {
		return 0
	}
	return m.sourceLines[renderedLine]
}

// renderDiff renders the diff content with syntax highlighting
// Returns the rendered content, total line count, and navigable line numbers
func (m *DiffViewModel) renderDiff() (string, int, []int) {
	startTime := time.Now()
	m.highlightTime = 0 // Reset accumulated highlight time

	if m.file == nil {
		return "", 0, nil
	}

	var b strings.Builder
	lineNum := 0             // Track current line number for cursor highlighting
	var navigableLines []int // Track which lines can have cursor (diff lines only)

	// Initialize comment, source line, and conversation tracking
	m.commentLines = make(map[int]int)
	m.sourceLines = make(map[int]int)
	m.lineConversationUUID = make(map[int]string)

	// Load conversations for this file from messaging interface
	var fileConversations []*critic.Conversation
	if m.messaging != nil {
		// Get the git-relative path
		gitPath := m.file.NewPath
		if m.file.IsDeleted {
			gitPath = m.file.OldPath
		}
		// Try to load conversations
		if convs, err := m.messaging.GetConversationsForFile(gitPath); err == nil {
			fileConversations = convs
		}
	}

	// Build a map of line number to conversation for quick lookup
	conversationsByLine := make(map[int]*critic.Conversation)
	for _, conv := range fileConversations {
		conversationsByLine[conv.LineNumber] = conv
	}

	filename := git.GitPathToDisplayPath(m.file.NewPath)
	if m.file.IsDeleted {
		filename = git.GitPathToDisplayPath(m.file.OldPath)
	}

	// Render file header
	header := fmt.Sprintf("📄 %s", filename)
	if m.file.IsNew {
		header += " (new)"
	} else if m.file.IsDeleted {
		header += " (deleted)"
	} else if m.file.IsRenamed {
		header = fmt.Sprintf("📄 %s → %s (renamed)",
			git.GitPathToDisplayPath(m.file.OldPath),
			git.GitPathToDisplayPath(m.file.NewPath))
	}

	b.WriteString(m.renderLineWithCursor(m.truncateToWidth(hunkHeaderStyle.Render(header)), lineNum))
	lineNum++
	b.WriteString("\n")
	b.WriteString(m.renderLineWithCursor(m.truncateToWidth(""), lineNum)) // Empty line
	lineNum++
	b.WriteString("\n")

	// Highlight files with appropriate background styles
	var oldFileDeleted map[int]string  // old file with deleted background
	var newFileAdded map[int]string    // new file with added background
	var newFileContext map[int]string  // new file with context background

	if m.highlightingEnabled {
		hlStart := time.Now()
		oldFileDeleted = m.highlightFullFileWithStyle(m.file, filename, true, highlight.GetDeletedStyle())
		newFileAdded = m.highlightFullFileWithStyle(m.file, filename, false, highlight.GetAddedStyle())
		newFileContext = m.highlightFullFileWithStyle(m.file, filename, false, highlight.GetContextStyle())
		m.highlightTime += time.Since(hlStart)
	}

	// Render each hunk
	for hunkIdx, hunk := range m.file.Hunks {
		// Render hunk header
		hunkHeader := fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
		if hunk.Header != "" {
			hunkHeader += " " + hunk.Header
		}
		b.WriteString(m.renderLineWithCursor(m.truncateToWidth(hunkHeaderStyle.Render(hunkHeader)), lineNum))
		lineNum++
		b.WriteString("\n")

		// Render hunk lines
		for _, line := range hunk.Lines {
			var highlighted string
			if m.highlightingEnabled {
				// Use pre-highlighted content with appropriate background
				switch line.Type {
				case ctypes.LineAdded:
					if hl, ok := newFileAdded[line.NewNum]; ok {
						highlighted = hl
					} else {
						highlighted = line.Content
					}
				case ctypes.LineDeleted:
					if hl, ok := oldFileDeleted[line.OldNum]; ok {
						highlighted = hl
					} else {
						highlighted = line.Content
					}
				case ctypes.LineContext:
					if hl, ok := newFileContext[line.NewNum]; ok {
						highlighted = hl
					} else {
						highlighted = line.Content
					}
				default:
					highlighted = line.Content
				}
			} else {
				highlighted = line.Content
			}

			lineStr := m.renderLine(line, highlighted, lineNum)
			b.WriteString(lineStr)
			// Add to navigable lines (only actual diff lines, not headers)
			navigableLines = append(navigableLines, lineNum)
			// Track mapping from rendered line to source line
			if line.NewNum > 0 {
				m.sourceLines[lineNum] = line.NewNum
			}
			lineNum++
			b.WriteString("\n")

			// Check if there's a conversation for this line
			if line.NewNum > 0 {
				if conv, exists := conversationsByLine[line.NewNum]; exists {
					// Render conversation preview
					commentLines := m.renderConversationPreview(conv, lineNum)
					for _, commentLine := range commentLines {
						b.WriteString(commentLine)
						// Track all comment lines for navigation
						m.commentLines[lineNum] = line.NewNum
						m.lineConversationUUID[lineNum] = conv.UUID
						// Add to navigable lines so user can select it
						navigableLines = append(navigableLines, lineNum)
						lineNum++
						b.WriteString("\n")
					}
				}
			}
		}

		// Add spacing between hunks
		if hunkIdx < len(m.file.Hunks)-1 {
			b.WriteString(m.renderLineWithCursor(m.truncateToWidth(""), lineNum))
			lineNum++
			b.WriteString("\n")
		}
	}

	result := b.String()
	totalTime := time.Since(startTime)
	renderTime := totalTime - m.highlightTime
	logger.Info("renderDiff: total=%v, highlight=%v, render=%v, lines=%d, navigable=%d",
		totalTime, m.highlightTime, renderTime, m.countLines(), len(navigableLines))

	return result, lineNum, navigableLines
}

// countLines returns the total number of lines in the current file
func (m *DiffViewModel) countLines() int {
	if m.file == nil {
		return 0
	}
	count := 0
	for _, hunk := range m.file.Hunks {
		count += len(hunk.Lines)
	}
	return count
}

// highlightFullFileWithStyle highlights a file with a specific chroma style
func (m *DiffViewModel) highlightFullFileWithStyle(file *ctypes.FileDiff, filename string, useOldVersion bool, style *chroma.Style) map[int]string {
	result := make(map[int]string)

	if file == nil {
		return result
	}

	// Get full file content from git
	var fullContent string
	var err error

	if useOldVersion {
		// Get old version (before changes)
		if file.IsNew {
			// New file has no old version
			return result
		}
		// For old version, always use hunk-based reconstruction since we don't know
		// which commit is the "old" version (it depends on diff mode: HEAD for unstaged,
		// HEAD~1 for last commit, merge-base for merge-base mode)
		return m.highlightFromHunks(file, filename, useOldVersion)
	} else {
		// Get new version (current working directory or HEAD depending on mode)
		if file.IsDeleted {
			// Deleted file has no new version
			return result
		}
		// First try working directory, then fall back to HEAD
		fullContent, err = git.GetFileContent(file.NewPath, "")
		if err != nil {
			// Fallback to HEAD for committed changes
			fullContent, err = git.GetFileContent(file.NewPath, "HEAD")
		}
	}

	if err != nil || fullContent == "" {
		// Fallback to hunk-based reconstruction if git fails
		return m.highlightFromHunks(file, filename, useOldVersion)
	}

	// Highlight the complete file with the specified style
	highlighted, err := m.highlighter.HighlightWithStyle(fullContent, filename, style)
	if err != nil {
		return result
	}

	// Split into lines and map by line number
	lines := strings.Split(highlighted, "\n")
	for i, line := range lines {
		lineNum := i + 1 // Line numbers are 1-based
		result[lineNum] = line
	}

	return result
}

// HighlightFullFileWithStyle highlights a full file with syntax highlighting.
// Exported for testing purposes.
func (m *DiffViewModel) HighlightFullFileWithStyle(file *ctypes.FileDiff, filename string, useOldVersion bool, style *chroma.Style) map[int]string {
	return m.highlightFullFileWithStyle(file, filename, useOldVersion, style)
}

// highlightFromHunks is a fallback that reconstructs partial file from hunks
func (m *DiffViewModel) highlightFromHunks(file *ctypes.FileDiff, filename string, useOldVersion bool) map[int]string {
	result := make(map[int]string)

	if file == nil || len(file.Hunks) == 0 {
		return result
	}

	// Reconstruct partial file content from hunks
	var lines []string
	var lineNumbers []int

	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			var includeThis bool
			var lineNum int

			if useOldVersion {
				includeThis = line.Type == ctypes.LineDeleted || line.Type == ctypes.LineContext
				lineNum = line.OldNum
			} else {
				includeThis = line.Type == ctypes.LineAdded || line.Type == ctypes.LineContext
				lineNum = line.NewNum
			}

			if includeThis && lineNum > 0 {
				lines = append(lines, line.Content)
				lineNumbers = append(lineNumbers, lineNum)
			}
		}
	}

	if len(lines) == 0 {
		return result
	}

	// Highlight all lines at once
	highlightedLines := m.highlighter.HighlightLines(lines, filename)

	// Map line numbers to highlighted content
	for i, lineNum := range lineNumbers {
		if i < len(highlightedLines) {
			result[lineNum] = highlightedLines[i]
		}
	}

	return result
}

// renderConversationPreview renders a conversation preview with all messages
func (m *DiffViewModel) renderConversationPreview(conv *critic.Conversation, startLineNum int) []string {
	// Build the complete text including all messages
	var allLines []string

	for i, msg := range conv.Messages {
		prefix := "human>"
		if msg.Author == critic.AuthorAI {
			prefix = "ai>"

			// Mark AI messages as read when they're displayed
			if msg.IsUnread && m.messaging != nil {
				if err := m.messaging.MarkAsRead(msg.UUID); err != nil {
					logger.Warn("Failed to mark AI message as read: %v", err)
				} else {
					logger.Debug("Marked AI message %s as read", msg.UUID)
				}
			}
		}

		// First message doesn't need prefix if it's human
		if i == 0 && msg.Author == critic.AuthorHuman {
			// Add each line of the root message
			msgLines := strings.Split(msg.Message, "\n")
			for _, line := range msgLines {
				allLines = append(allLines, renderMarkdown(line))
			}
		} else {
			// Add each line of the reply with the prefix
			replyLines := strings.Split(msg.Message, "\n")
			for _, line := range replyLines {
				allLines = append(allLines, prefix+" "+renderMarkdown(line))
			}
		}
	}

	// Check if the conversation is resolved
	if conv.Status == critic.StatusResolved {
		// Prepend "(Resolved)" to the first line
		if len(allLines) > 0 {
			allLines[0] = "\x1b[3m(Resolved)\x1b[23m " + allLines[0]
		}
	}

	// ANSI color codes - light blue background with black text (like status bar)
	const lightBlueBg = "\x1b[48;2;107;149;216m" // #6B95D8 - same as status bar
	const blackFg = "\x1b[38;5;0m"               // Black text
	const grayFg = "\x1b[38;5;240m"              // Gray text for separator
	const reset = "\x1b[0m"

	// Calculate the total lines including blank lines before/after
	// Structure: content lines, separator line with hotkeys (when selected)
	totalContentLines := len(allLines)
	maxLines := 6

	// Calculate total block size to determine if cursor is inside
	linesToRender := totalContentLines
	hasMore := totalContentLines > maxLines

	// Block size calculation: separator + content + separator
	blockSize := 1 + linesToRender + 1 // top separator + content + bottom separator
	if hasMore {
		blockSize = 1 + maxLines + 1 + 1 // top separator + truncated content + "more" indicator + bottom separator
	}

	cursorInBlock := m.focused && m.cursorLine >= startLineNum && m.cursorLine < startLineNum+blockSize

	// If cursor is in the block, show full content
	if cursorInBlock {
		linesToRender = totalContentLines
		hasMore = false
		blockSize = 1 + linesToRender + 1 // top separator + full content + bottom separator
	} else {
		if linesToRender > maxLines {
			linesToRender = maxLines
		}
	}

	// Build result: content lines, separator with hotkeys
	var result []string

	// Helper to create a content line (black text on light blue)
	createContentLine := func(text string, lineNum int) string {
		content := " " + text
		visibleWidth := lipgloss.Width(content)
		availableWidth := m.width

		var processed string
		if visibleWidth > availableWidth {
			processed = truncateANSI(content, availableWidth)
		} else {
			padding := strings.Repeat(" ", availableWidth-visibleWidth)
			processed = content + padding
		}
		styled := lightBlueBg + blackFg + processed + reset
		return m.renderLineWithCursor(styled, lineNum)
	}

	// Helper to create the separator line with optional hotkeys
	createSeparatorLine := func(text string, lineNum int) string {
		// Separator is a line of dashes with optional centered text
		availableWidth := m.width
		if text == "" {
			line := strings.Repeat("─", availableWidth)
			styled := grayFg + line + reset
			return m.renderLineWithCursor(styled, lineNum)
		}
		// Center the text in the separator
		textLen := lipgloss.Width(text)
		leftDashes := (availableWidth - textLen - 2) / 2
		rightDashes := availableWidth - textLen - 2 - leftDashes
		if leftDashes < 0 {
			leftDashes = 0
		}
		if rightDashes < 0 {
			rightDashes = 0
		}
		line := strings.Repeat("─", leftDashes) + " " + text + " " + strings.Repeat("─", rightDashes)
		styled := grayFg + line + reset
		return m.renderLineWithCursor(styled, lineNum)
	}

	currentLine := startLineNum

	// Add top separator line (plain, no hotkeys)
	result = append(result, createSeparatorLine("", currentLine))
	currentLine++

	// Add content lines
	for i := 0; i < linesToRender; i++ {
		result = append(result, createContentLine(allLines[i], currentLine))
		currentLine++
	}

	// Add "more lines" indicator if truncated
	if hasMore {
		moreCount := totalContentLines - maxLines
		indicatorText := fmt.Sprintf("(%d more lines)", moreCount)
		result = append(result, createContentLine(indicatorText, currentLine))
		currentLine++
	}

	// Add separator line - with hotkeys if cursor is in the block
	var separatorText string
	if cursorInBlock {
		separatorText = "[R]esolve • [Enter] reply"
		if conv.Status == critic.StatusResolved {
			separatorText = "[R] unresolve • [Enter] reply"
		}
	}
	result = append(result, createSeparatorLine(separatorText, currentLine))

	return result
}

// renderMarkdown applies simple markdown formatting to text
// Supports **bold** and _underline_
func renderMarkdown(text string) string {
	const bold = "\x1b[1m"
	const underline = "\x1b[4m"
	const reset = "\x1b[0m"

	result := text

	// Handle **bold** - replace **text** with bold ANSI
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	result = boldRegex.ReplaceAllString(result, bold+"$1"+reset)

	// Handle _underline_ - replace _text_ with underline ANSI
	// Use word boundaries to avoid matching variable_names_like_this
	underlineRegex := regexp.MustCompile(`\b_([^_]+)_\b`)
	result = underlineRegex.ReplaceAllString(result, underline+"$1"+reset)

	return result
}

// renderLine renders a single diff line with pre-highlighted content
func (m *DiffViewModel) renderLine(line *ctypes.Line, highlightedContent string, currentLineNum int) string {
	// Build line number prefix: " 123 + "
	var lineNum int
	var indicator string
	switch line.Type {
	case ctypes.LineAdded:
		lineNum = line.NewNum
		indicator = "+"
	case ctypes.LineDeleted:
		lineNum = line.OldNum
		indicator = "-"
	case ctypes.LineContext:
		lineNum = line.NewNum
		indicator = " "
	}

	prefix := fmt.Sprintf("%4d %s ", lineNum, indicator)

	// Combine prefix with pre-highlighted content
	fullLine := prefix + highlightedContent

	// Chroma provides backgrounds but may reset between tokens
	// We need to ensure background spans the full line
	styled := m.ensureFullLineBackground(fullLine, line.Type)

	// Apply cursor highlighting if this is the active line
	return m.renderLineWithCursor(styled, currentLineNum)
}

// renderLineWithCursor applies cursor highlighting if this is the active line
func (m *DiffViewModel) renderLineWithCursor(content string, currentLineNum int) string {
	// Only show cursor when pane is focused and line is navigable
	if m.focused && currentLineNum == m.cursorLine && m.isNavigableLine(currentLineNum) {
		// Apply full reverse highlighting
		return lipgloss.NewStyle().Reverse(true).Render(content)
	}
	return content
}

// isNavigableLine checks if a line number is navigable (can have cursor)
func (m *DiffViewModel) isNavigableLine(lineNum int) bool {
	for _, navLine := range m.navigableLines {
		if navLine == lineNum {
			return true
		}
	}
	return false
}

// truncateToWidth truncates or pads a line to exactly match viewport width
func (m *DiffViewModel) truncateToWidth(line string) string {
	visibleWidth := lipgloss.Width(line)
	if visibleWidth > m.width {
		return truncateANSI(line, m.width)
	} else if visibleWidth < m.width {
		// Pad to full width - need to preserve the last background color for padding
		// Extract the last background color from the line (if any)
		lastBg := extractLastBackground(line)
		padding := strings.Repeat(" ", m.width - visibleWidth)
		if lastBg != "" {
			// Apply the background color to padding spaces
			return line + lastBg + padding + "\x1b[0m"
		}
		return line + padding
	}
	return line
}

// ensureFullLineBackground strips chroma's backgrounds and applies our own for full width
func (m *DiffViewModel) ensureFullLineBackground(line string, lineType ctypes.LineType) string {
	// Strip all background codes from chroma
	cleaned := stripAllStyleCodes(line)

	// Get the appropriate background color for this line type
	var bgCode string
	switch lineType {
	case ctypes.LineAdded:
		bgCode = "\x1b[48;5;22m" // Dark green in 256-color
	case ctypes.LineDeleted:
		bgCode = "\x1b[48;5;52m" // Dark red in 256-color
	default:
		bgCode = "\x1b[48;5;0m" // Black in 256-color
	}

	// Calculate width and pad if needed
	visibleWidth := lipgloss.Width(cleaned)
	if visibleWidth > m.width {
		cleaned = truncateANSI(cleaned, m.width)
		visibleWidth = m.width
	}

	padding := ""
	if visibleWidth < m.width {
		padding = strings.Repeat(" ", m.width - visibleWidth)
	}

	// Wrap entire line with background: bg + content + padding + reset
	return bgCode + cleaned + padding + "\x1b[0m"
}

// extractLastBackground finds the last background color code in a string
func extractLastBackground(s string) string {
	// Look for background color codes: \x1b[48;... or \x1b[4X where X is 0-9
	bgRegex := regexp.MustCompile(`\x1b\[(?:48;[0-9;]+|4[0-9])m`)
	matches := bgRegex.FindAllString(s, -1)
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}
	return ""
}

// applyLineBackgroundWithStyle applies background using lipgloss style (supports adaptive colors)
func (m *DiffViewModel) applyLineBackgroundWithStyle(line string, style lipgloss.Style) string {
	// Strip all style codes except foreground colors
	cleaned := stripAllStyleCodes(line)

	// Calculate visible width
	visibleWidth := lipgloss.Width(cleaned)

	// Truncate if too long, pad if too short
	var processed string
	if visibleWidth > m.width {
		processed = truncateANSI(cleaned, m.width)
	} else {
		paddingWidth := m.width - visibleWidth
		processed = cleaned + strings.Repeat(" ", paddingWidth)
	}

	// Use lipgloss style to render with proper background (handles adaptive colors)
	return style.Width(m.width).Render(processed)
}

// applyLineBackground wraps a line with background color spanning full width (legacy)
func (m *DiffViewModel) applyLineBackground(line string, bgColor string) string {
	// Strip any background ANSI codes and reset codes from syntax highlighting
	cleaned := stripBackgroundCodes(line)
	cleaned = stripResetCodes(cleaned)

	// Note: tabs are already expanded in renderLine before this function is called

	// Calculate visible width (accounting for ANSI codes and expanded tabs)
	visibleWidth := lipgloss.Width(cleaned)

	// Truncate if too long, pad if too short
	var processed string
	if visibleWidth > m.width {
		// Line is too long - truncate it to exact width
		processed = truncateANSI(cleaned, m.width)
	} else {
		// Add padding to reach exact width
		paddingWidth := m.width - visibleWidth
		processed = cleaned + strings.Repeat(" ", paddingWidth)
	}

	// Build final string: bg + content + full reset
	// Now that we've stripped all resets, the background will span the entire line
	return bgColor + processed + "\x1b[0m"
}

// truncateANSI truncates a string with ANSI codes to a specific visible width
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// resolveCommentAtCursor toggles the resolved status of the comment at the current cursor position
func (m *DiffViewModel) resolveCommentAtCursor() tea.Cmd {
	// Check if cursor is on a comment line
	if isComment, _ := m.IsCommentLine(m.cursorLine); isComment {
		// Get conversation UUID for this line
		uuid := m.GetConversationUUIDAtLine(m.cursorLine)
		if uuid != "" && m.messaging != nil {
			// Get the current conversation to check its status
			conv, err := m.messaging.GetFullConversation(uuid)
			if err != nil {
				logger.Error("Failed to get conversation: %v", err)
				return nil
			}

			// Toggle the status
			if conv.Status == critic.StatusResolved {
				err = m.messaging.MarkAsUnresolved(uuid)
				if err != nil {
					logger.Error("Failed to mark conversation as unresolved: %v", err)
					return nil
				}
				logger.Info("Marked comment %s as unresolved", uuid)
			} else {
				err = m.messaging.MarkAsResolved(uuid)
				if err != nil {
					logger.Error("Failed to mark conversation as resolved: %v", err)
					return nil
				}
				logger.Info("Marked comment %s as resolved", uuid)
			}

			// Refresh the view to show the updated status
			return m.RefreshFile()
		}
	}

	return nil
}
