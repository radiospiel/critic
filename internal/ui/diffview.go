package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.15b.it/eno/critic/internal/comments"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/highlight"
	"git.15b.it/eno/critic/internal/logger"
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
	focused             bool          // Whether this pane is focused
	navigableLines      []int         // Line numbers that can have cursor (diff lines only)
	commentManager      *comments.FileManager
	commentLines        map[int]int   // Maps rendered line number to source line number for comment lines
	sourceLines         map[int]int   // Maps rendered line number to source line number for all diff lines
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
				}
			case "down", "j":
				if m.moveCursorDown() {
					m.ensureCursorVisible()
					return m.refreshContent()
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
				m.viewport.GotoTop()
				// Reset cursor to first navigable line
				if len(m.navigableLines) > 0 {
					m.cursorLine = m.navigableLines[0]
				}
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

	// Clear cache to force re-render
	m.cachedFile = nil

	// Re-render with highlighting if enabled
	if m.highlightingEnabled {
		return m.renderDiffAsync(m.file)
	}

	// Otherwise render immediately
	content, totalLines, navigableLines := m.renderDiff()
	m.cachedContent = content
	m.totalLines = totalLines
	m.navigableLines = navigableLines
	if m.ready {
		m.viewport.SetContent(m.cachedContent)
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
func (m *DiffViewModel) ScrollPageDown() tea.Cmd {
	if !m.ready || len(m.navigableLines) == 0 {
		return nil
	}
	scrollAmount := m.viewport.Height - 3
	if scrollAmount < 1 {
		scrollAmount = 1
	}

	// Calculate new offset
	newOffset := m.viewport.YOffset + scrollAmount
	maxOffset := m.totalLines - m.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}
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
func (m *DiffViewModel) ScrollPageUp() tea.Cmd {
	if !m.ready || len(m.navigableLines) == 0 {
		return nil
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

// SetCommentManager sets the comment manager for loading comments
func (m *DiffViewModel) SetCommentManager(cm *comments.FileManager) {
	m.commentManager = cm
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

	// Initialize comment and source line tracking
	m.commentLines = make(map[int]int)
	m.sourceLines = make(map[int]int)

	// Load comments for this file if comment manager is available
	var fileComments *ctypes.CriticFile
	if m.commentManager != nil {
		// Get the git-relative path
		gitPath := m.file.NewPath
		if m.file.IsDeleted {
			gitPath = m.file.OldPath
		}
		// Try to load comments
		if loaded, err := m.commentManager.LoadComments(gitPath); err == nil {
			fileComments = loaded
		}
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

			// Check if there's a comment for this line
			if fileComments != nil && line.NewNum > 0 {
				if comment, exists := fileComments.Comments[line.NewNum]; exists && len(comment.Lines) > 0 {
					// Render up to 6 lines of the comment preview
					commentLines := m.renderCommentPreview(comment.Lines, lineNum)
					for i, commentLine := range commentLines {
						b.WriteString(commentLine)
						// Track the first comment line for navigation
						if i == 0 {
							m.commentLines[lineNum] = line.NewNum
						}
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

// renderCommentPreview renders up to 6 lines of a comment preview with dark text on black background
// Each line starts with a half-width block character in yellow/gold color
func (m *DiffViewModel) renderCommentPreview(commentLines []string, startLineNum int) []string {
	// Limit to 6 lines
	maxLines := 6
	linesToRender := len(commentLines)
	if linesToRender > maxLines {
		linesToRender = maxLines
	}

	result := make([]string, linesToRender)

	// ANSI color codes
	const yellowFg = "\x1b[38;5;220m"     // Yellow/gold foreground for block
	const darkFg = "\x1b[38;5;250m"       // Light gray text
	const darkYellowBg = "\x1b[48;5;58m"  // Dark yellowish background
	const reset = "\x1b[0m"

	// Half-width block characters
	const leftHalfBlock = "▌"
	const rightHalfBlock = "▐"

	for i := 0; i < linesToRender; i++ {
		// Start with colored half-block + space
		prefix := yellowFg + leftHalfBlock + reset + " "

		// Combine prefix and comment text
		content := prefix + commentLines[i]

		// Calculate visible width (accounting for the right block we'll add)
		visibleWidth := lipgloss.Width(content)

		// Reserve space for the right half-block
		availableWidth := m.width - 1 // -1 for right half-block

		// Truncate or pad to match viewport width minus the right block
		var processed string
		if visibleWidth > availableWidth {
			processed = truncateANSI(content, availableWidth)
		} else {
			padding := strings.Repeat(" ", availableWidth-visibleWidth)
			processed = content + padding
		}

		// Add right half-block at the end
		processed = processed + yellowFg + rightHalfBlock + reset

		// Apply dark yellowish background and light text
		styled := darkYellowBg + darkFg + processed + reset

		// Apply cursor highlighting if this is the active line
		currentLineNum := startLineNum + i
		result[i] = m.renderLineWithCursor(styled, currentLineNum)
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
