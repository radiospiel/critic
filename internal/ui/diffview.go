package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.15b.it/eno/critic/internal/config"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/highlight"
	"git.15b.it/eno/critic/pkg/critic"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/simple-go/logger"
	"git.15b.it/eno/critic/teapot"
	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FilterMode represents the current file/hunk filter mode
type FilterMode int

const (
	// FilterModeNone shows all files and hunks (default)
	FilterModeNone FilterMode = iota
	// FilterModeWithComments shows only files with comments, and only hunks with comments
	FilterModeWithComments
	// FilterModeWithUnresolved shows only files with unresolved comments, and only hunks with unresolved comments
	FilterModeWithUnresolved
)

// DiffViewModel represents the diff viewer pane
type DiffViewModel struct {
	file                 *ctypes.FileDiff
	viewport             viewport.Model
	width                int
	height               int
	ready                bool
	highlighter          *highlight.Highlighter
	cachedContent        string
	cachedFile           *ctypes.FileDiff
	highlightingEnabled  bool
	highlightTime        time.Duration // Accumulated syntax highlighting time
	cursorLine           int           // Current active line (0-based)
	totalLines           int           // Total number of lines in rendered diff
	focused              bool          // Whether this pane is focused
	navigableLines       []int         // Line numbers that can have cursor (diff lines only)
	messaging            critic.Messaging
	commentLines         map[int]int    // Maps rendered line number to source line number for comment lines
	sourceLines          map[int]int    // Maps rendered line number to source line number for all diff lines
	preserveCursorLine   int            // Source line to restore cursor to after refresh (0 = don't preserve)
	lineConversationUUID map[int]string // Maps rendered line number to conversation UUID
	gotoBottomOnLoad     bool           // If true, go to bottom after next file load
	filterMode           FilterMode     // Current filter mode for hunk filtering
	animationTicker      *AnimationTicker // Animation ticker for conversation states

	// Widget-based rendering
	diffWidget           *DiffViewWidget // The widget that renders the vertical layout of hunks

	// Cached syntax highlighting (expensive to compute, reused across renders)
	cachedHighlightFile  *ctypes.FileDiff   // File the cached highlights are for
	cachedOldFileDeleted map[int]string     // Cached highlighted deleted lines
	cachedNewFileAdded   map[int]string     // Cached highlighted added lines
	cachedNewFileContext map[int]string     // Cached highlighted context lines
}

// NewDiffViewModel creates a new diff viewer model
func NewDiffViewModel() DiffViewModel {
	return DiffViewModel{
		highlighter:         highlight.NewHighlighter(),
		highlightingEnabled: true, // Default to enabled
		diffWidget:          NewDiffViewWidget(),
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

// RefreshAnimations re-renders the diff content to update animation frames.
// This uses cached syntax highlighting so it's fast - only the widget rendering
// and animation overlay are recomputed.
func (m *DiffViewModel) RefreshAnimations() {
	if m.file == nil || m.animationTicker == nil {
		return
	}
	// Only refresh if there are active animations
	if !m.diffWidget.HasAnimations() {
		return
	}
	content, totalLines, navigableLines := m.renderDiff()
	m.cachedContent = content
	m.totalLines = totalLines
	m.navigableLines = navigableLines
	if m.ready {
		m.viewport.SetContent(m.cachedContent)
	}
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

// SetAnimationTicker sets the animation ticker for conversation state animations
func (m *DiffViewModel) SetAnimationTicker(ticker *AnimationTicker) {
	m.animationTicker = ticker
}

// SetFilterMode sets the filter mode for hunk filtering
func (m *DiffViewModel) SetFilterMode(mode FilterMode) {
	if m.filterMode != mode {
		m.filterMode = mode
		// Clear cache to force re-render with new filter
		m.cachedFile = nil
	}
}

// filterHunks filters hunks based on the current filter mode
func (m *DiffViewModel) filterHunks(hunks []*ctypes.Hunk, conversationsByLine map[int]*critic.Conversation) []*ctypes.Hunk {
	// If no filter, return all hunks
	if m.filterMode == FilterModeNone {
		return hunks
	}

	var filtered []*ctypes.Hunk
	for _, hunk := range hunks {
		if m.hunkMatchesFilter(hunk, conversationsByLine) {
			filtered = append(filtered, hunk)
		}
	}

	logger.Info("filterHunks: mode=%d, hunks=%d->%d, conversations=%d",
		m.filterMode, len(hunks), len(filtered), len(conversationsByLine))

	return filtered
}

// hunkMatchesFilter checks if a hunk should be included based on the current filter mode
func (m *DiffViewModel) hunkMatchesFilter(hunk *ctypes.Hunk, conversationsByLine map[int]*critic.Conversation) bool {
	for _, line := range hunk.Lines {
		if line.NewNum > 0 {
			if conv, exists := conversationsByLine[line.NewNum]; exists {
				switch m.filterMode {
				case FilterModeWithComments:
					// Any comment matches
					return true
				case FilterModeWithUnresolved:
					// Only unresolved comments match
					if conv.Status != critic.StatusResolved {
						return true
					}
				}
			}
		}
	}
	return false
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

// renderDiff renders the diff content using widget-based vertical layout.
// Each hunk is a vertical layout containing lines and their associated comments.
// Returns the rendered content, total line count, and navigable line numbers.
func (m *DiffViewModel) renderDiff() (string, int, []int) {
	startTime := time.Now()
	m.highlightTime = 0 // Reset accumulated highlight time

	if m.file == nil {
		return "", 0, nil
	}

	// Initialize tracking maps
	m.commentLines = make(map[int]int)
	m.sourceLines = make(map[int]int)
	m.lineConversationUUID = make(map[int]string)

	// Load conversations for this file
	conversationsByLine := m.loadConversationsForFile()

	filename := git.GitPathToDisplayPath(m.file.NewPath)
	if m.file.IsDeleted {
		filename = git.GitPathToDisplayPath(m.file.OldPath)
	}

	// Build highlighted content maps (cached - only recompute when file changes)
	var oldFileDeleted, newFileAdded, newFileContext map[int]string
	if m.highlightingEnabled {
		// Check if we can reuse cached highlights
		if m.cachedHighlightFile == m.file && m.cachedOldFileDeleted != nil {
			// Reuse cached highlights
			oldFileDeleted = m.cachedOldFileDeleted
			newFileAdded = m.cachedNewFileAdded
			newFileContext = m.cachedNewFileContext
		} else {
			// Compute and cache highlights
			hlStart := time.Now()
			oldFileDeleted = m.highlightFullFileWithStyle(m.file, filename, true, highlight.GetDeletedStyle())
			newFileAdded = m.highlightFullFileWithStyle(m.file, filename, false, highlight.GetAddedStyle())
			newFileContext = m.highlightFullFileWithStyle(m.file, filename, false, highlight.GetContextStyle())
			m.highlightTime += time.Since(hlStart)

			// Cache for reuse
			m.cachedHighlightFile = m.file
			m.cachedOldFileDeleted = oldFileDeleted
			m.cachedNewFileAdded = newFileAdded
			m.cachedNewFileContext = newFileContext
		}
	}

	// Configure the widget - set filter mode and animation BEFORE SetFile
	// since SetFile calls rebuildHunks which uses these values
	m.diffWidget.SetAnimationTicker(m.animationTicker)
	m.diffWidget.SetFilterMode(m.filterMode)
	m.diffWidget.SetFile(m.file, conversationsByLine, oldFileDeleted, newFileAdded, newFileContext)

	// Calculate content height for the widget
	contentHeight := m.calculateContentHeight(conversationsByLine)

	// Create buffer with appropriate dimensions
	bufferWidth := m.width
	if bufferWidth <= 0 {
		bufferWidth = 80 // default width
	}
	bufferHeight := contentHeight
	if bufferHeight <= 0 {
		bufferHeight = 1
	}

	// Create and render to buffer
	buffer := teapot.NewBuffer(bufferWidth, bufferHeight)
	subBuf := buffer.Sub(teapot.Rect{X: 0, Y: 0, Width: bufferWidth, Height: bufferHeight})

	// Render the widget - this creates the vertical layout of hunks
	m.diffWidget.Render(subBuf)

	// Build line mappings and navigable lines from the widget structure
	navigableLines := m.buildLineMappingsFromWidget(conversationsByLine)

	// Update widget selection for hotkey display in comments
	m.diffWidget.SetSelectedRow(m.cursorLine)

	// Re-render with the updated selection (for hotkey display)
	buffer.Clear()
	m.diffWidget.Render(subBuf)

	// Apply animation overlays (fills in animation content after placeholder render)
	if m.diffWidget.HasAnimations() {
		m.diffWidget.RenderAnimationOverlays(buffer)
	}

	// Apply selection overlay - invert the current line if focused and navigable
	if m.focused && m.isNavigableLine(m.cursorLine) {
		buffer.InvertRow(m.cursorLine)
	}

	// Convert buffer to string
	result := buffer.String()

	totalTime := time.Since(startTime)
	renderTime := totalTime - m.highlightTime
	logger.Info("renderDiff (widget): total=%.1fms, highlight=%.1fms, render=%.1fms, lines=%d, navigable=%d",
		float64(totalTime.Microseconds())/1000.0,
		float64(m.highlightTime.Microseconds())/1000.0,
		float64(renderTime.Microseconds())/1000.0,
		m.countLines(), len(navigableLines))

	return result, contentHeight, navigableLines
}

// loadConversationsForFile loads and returns conversations indexed by line number
func (m *DiffViewModel) loadConversationsForFile() map[int]*critic.Conversation {
	conversationsByLine := make(map[int]*critic.Conversation)
	if m.messaging == nil || m.file == nil {
		return conversationsByLine
	}

	gitPath := m.file.NewPath
	if m.file.IsDeleted {
		gitPath = m.file.OldPath
	}

	if convs, err := m.messaging.GetConversationsForFile(gitPath); err == nil {
		for _, conv := range convs {
			conversationsByLine[conv.LineNumber] = conv
		}
	}
	return conversationsByLine
}

// calculateContentHeight calculates the total height needed for the content
func (m *DiffViewModel) calculateContentHeight(conversationsByLine map[int]*critic.Conversation) int {
	if m.file == nil {
		return 0
	}

	height := 2 // File header

	hunksToRender := m.filterHunks(m.file.Hunks, conversationsByLine)
	for hunkIdx, hunk := range hunksToRender {
		height++ // Hunk header
		for _, line := range hunk.Lines {
			height++ // Line
			if line.NewNum > 0 {
				if conv, exists := conversationsByLine[line.NewNum]; exists {
					// Comment height: separator + content + separator
					height += calculateCommentHeight(conv)
				}
			}
		}
		if hunkIdx < len(hunksToRender)-1 {
			height++ // Spacing
		}
	}
	return height
}

// buildLineMappingsFromWidget builds the line mappings from the widget structure
func (m *DiffViewModel) buildLineMappingsFromWidget(conversationsByLine map[int]*critic.Conversation) []int {
	var navigableLines []int

	if m.file == nil {
		return navigableLines
	}

	lineNum := 2 // Start after file header
	hunksToRender := m.filterHunks(m.file.Hunks, conversationsByLine)

	for hunkIdx, hunk := range hunksToRender {
		lineNum++ // Hunk header

		for _, line := range hunk.Lines {
			// Track this line as navigable
			navigableLines = append(navigableLines, lineNum)
			if line.NewNum > 0 {
				m.sourceLines[lineNum] = line.NewNum
			}
			lineNum++

			// Check for comment
			if line.NewNum > 0 {
				if conv, exists := conversationsByLine[line.NewNum]; exists {
					commentHeight := calculateCommentHeight(conv)

					// Track all comment lines
					for i := 0; i < commentHeight; i++ {
						m.commentLines[lineNum+i] = line.NewNum
						m.lineConversationUUID[lineNum+i] = conv.UUID
						navigableLines = append(navigableLines, lineNum+i)
					}
					lineNum += commentHeight
				}
			}
		}

		if hunkIdx < len(hunksToRender)-1 {
			lineNum++ // Spacing
		}
	}

	return navigableLines
}

// updateWidgetSelection updates the widget's selection based on cursorLine
func (m *DiffViewModel) updateWidgetSelection() {
	if m.diffWidget == nil {
		return
	}

	// Update the widget's selected row to match cursor position
	m.diffWidget.SetSelectedRow(m.cursorLine)
}

// findSelectableItemForLine finds the selectable item index for a given line number
func (m *DiffViewModel) findSelectableItemForLine(lineNum int) int {
	if m.file == nil || m.diffWidget == nil {
		return -1
	}

	conversationsByLine := m.loadConversationsForFile()
	currentLine := 2 // Start after file header
	itemIdx := 0
	hunksToRender := m.filterHunks(m.file.Hunks, conversationsByLine)

	for hunkIdx, hunk := range hunksToRender {
		currentLine++ // Hunk header

		for _, line := range hunk.Lines {
			if currentLine == lineNum {
				return itemIdx
			}
			currentLine++
			itemIdx++

			if line.NewNum > 0 {
				if conv, exists := conversationsByLine[line.NewNum]; exists {
					commentHeight := calculateCommentHeight(conv)

					// Check if lineNum falls within comment
					if lineNum >= currentLine && lineNum < currentLine+commentHeight {
						return itemIdx
					}
					currentLine += commentHeight
					itemIdx++
				}
			}
		}

		if hunkIdx < len(hunksToRender)-1 {
			currentLine++ // Spacing
		}
	}

	return -1
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
		prefix := "You" // Human
		if msg.Author == critic.AuthorAI {
			prefix = "AI"

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
			for j, line := range replyLines {
				if j == 0 {
					allLines = append(allLines, fmt.Sprintf("%s: %s", prefix, renderMarkdown(line)))
				} else {
					// Indent continuation lines (align with message text after "You: " or "AI: ")
					indent := strings.Repeat(" ", len(prefix)+2)
					allLines = append(allLines, indent+renderMarkdown(line))
				}
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

	// Available width for content (full width, no animation prefix on content)
	availableWidth := m.width

	// Wrap long lines instead of truncating
	var wrappedLines []string
	for _, line := range allLines {
		wrapped := wrapLine(line, availableWidth-2) // -2 for padding on sides
		wrappedLines = append(wrappedLines, wrapped...)
	}
	allLines = wrappedLines

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

	// Helper to create a content line (black text on light blue) - no animation prefix
	createContentLine := func(text string, lineNum int) string {
		content := " " + text
		visibleWidth := lipgloss.Width(content)

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

	// Helper to create the top separator line with snake animation
	createTopSeparatorLine := func(lineNum int) string {
		// Get snake animation frame (12 chars)
		var animFrame string
		if m.animationTicker != nil {
			animFrame = m.animationTicker.GetSeparatorFrame()
		} else {
			animFrame = "○○○○○○○○○○○○" // fallback static snake
		}

		// Calculate dashes needed to fill the rest of the line
		animWidth := 12 // snake animation is 12 chars
		dashesNeeded := availableWidth - animWidth - 1 // -1 for space after animation
		if dashesNeeded < 0 {
			dashesNeeded = 0
		}
		line := animFrame + " " + strings.Repeat("─", dashesNeeded)
		styled := grayFg + line + reset
		return m.renderLineWithCursor(styled, lineNum)
	}

	// Helper to create the bottom separator line with optional hotkeys
	createBottomSeparatorLine := func(text string, lineNum int) string {
		// Separator is a line of dashes with optional centered text
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

	// Add top separator line with snake animation
	result = append(result, createTopSeparatorLine(currentLine))
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
	result = append(result, createBottomSeparatorLine(separatorText, currentLine))

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

// wrapLine wraps a line of text (possibly with ANSI codes) to fit within maxWidth.
// Returns a slice of wrapped lines, preserving ANSI codes across line breaks.
func wrapLine(line string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{line}
	}

	// Calculate visible width
	visibleWidth := lipgloss.Width(line)
	if visibleWidth <= maxWidth {
		return []string{line}
	}

	// Simple word-wrapping: split on spaces and accumulate words
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{line}
	}

	var result []string
	var currentLine strings.Builder
	currentWidth := 0

	for i, word := range words {
		wordWidth := lipgloss.Width(word)

		if currentWidth == 0 {
			// First word on line
			currentLine.WriteString(word)
			currentWidth = wordWidth
		} else if currentWidth+1+wordWidth <= maxWidth {
			// Word fits on current line
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
			currentWidth += 1 + wordWidth
		} else {
			// Word doesn't fit - start new line
			result = append(result, currentLine.String())
			currentLine.Reset()
			// Add indent for continuation (2 spaces)
			currentLine.WriteString("  ")
			currentLine.WriteString(word)
			currentWidth = 2 + wordWidth
		}

		// Handle very long words that exceed maxWidth by themselves
		if i == 0 && wordWidth > maxWidth {
			// Just add it as-is, it will get truncated later
			result = append(result, currentLine.String())
			currentLine.Reset()
			currentWidth = 0
		}
	}

	// Add remaining line
	if currentLine.Len() > 0 {
		result = append(result, currentLine.String())
	}

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
		padding := strings.Repeat(" ", m.width-visibleWidth)
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
		padding = strings.Repeat(" ", m.width-visibleWidth)
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
