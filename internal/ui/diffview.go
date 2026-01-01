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
	commentInput        CommentInputModel
	commentStorage      *comments.Storage
	fileComments        map[int]*comments.Comment // Comments for current file (keyed by line number)
	lineToFileLineNum   map[int]int               // Map from display line number to actual file line number
}

// NewDiffViewModel creates a new diff viewer model
func NewDiffViewModel() DiffViewModel {
	storage, err := comments.NewStorage()
	if err != nil {
		logger.Error("Failed to create comment storage: %v", err)
		storage = nil
	}

	return DiffViewModel{
		highlighter:         highlight.NewHighlighter(),
		highlightingEnabled: true, // Default to enabled
		commentInput:        NewCommentInputModel(),
		commentStorage:      storage,
		fileComments:        make(map[int]*comments.Comment),
		lineToFileLineNum:   make(map[int]int),
	}
}

// Init initializes the diff view model
func (m DiffViewModel) Init() tea.Cmd {
	return nil
}

// Update updates the diff view model
func (m *DiffViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	// If comment input is visible, route messages to it
	if m.commentInput.IsVisible() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// All key messages go to comment input when it's visible
			m.commentInput, cmd = m.commentInput.Update(msg)
			return cmd
		case CommentSavedMsg:
			// Save the comment
			if m.commentStorage != nil && m.file != nil {
				filePath := m.file.NewPath
				if filePath == "" {
					filePath = m.file.OldPath
				}
				err := m.commentStorage.SaveComment(filePath, msg.LineNumber, msg.Content)
				if err != nil {
					logger.Error("Failed to save comment: %v", err)
				} else {
					logger.Info("Comment saved for line %d", msg.LineNumber)
					// Reload comments for this file
					m.loadCommentsForCurrentFile()
				}
			}
			m.commentInput.Hide()
			return m.refreshContent()
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.ready {
			switch msg.String() {
			case "enter":
				// Open comment input for current line
				if m.focused && m.commentStorage != nil {
					fileLineNum, ok := m.lineToFileLineNum[m.cursorLine]
					if ok {
						filePath := m.file.NewPath
						if filePath == "" {
							filePath = m.file.OldPath
						}

						// Check if there's an existing comment
						if comment, exists := m.fileComments[fileLineNum]; exists {
							// Edit existing comment
							m.commentInput.ShowEdit(fileLineNum, filePath, comment.Content)
						} else {
							// Create new comment
							m.commentInput.Show(fileLineNum, filePath)
						}
						return nil
					}
				}

			case "d":
				// Delete comment at current line (if exists)
				if m.focused && m.commentStorage != nil {
					fileLineNum, ok := m.lineToFileLineNum[m.cursorLine]
					if ok {
						if _, exists := m.fileComments[fileLineNum]; exists {
							filePath := m.file.NewPath
							if filePath == "" {
								filePath = m.file.OldPath
							}
							err := m.commentStorage.DeleteComment(filePath, fileLineNum)
							if err != nil {
								logger.Error("Failed to delete comment: %v", err)
							} else {
								logger.Info("Comment deleted for line %d", fileLineNum)
								// Reload comments
								m.loadCommentsForCurrentFile()
								return m.refreshContent()
							}
						}
					}
				}

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
	// If comment input is visible, show it as an overlay
	if m.commentInput.IsVisible() {
		// Show comment input overlay on top of diff view
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.commentInput.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.AdaptiveColor{Light: "#000", Dark: "#000"}),
		)
	}

	return m.renderBaseView()
}

// renderBaseView renders the base diff view without overlays
func (m DiffViewModel) renderBaseView() string {
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

	// Load comments for this file
	m.loadCommentsForCurrentFile()

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

// loadCommentsForCurrentFile loads comments for the currently displayed file
func (m *DiffViewModel) loadCommentsForCurrentFile() {
	if m.file == nil || m.commentStorage == nil {
		m.fileComments = make(map[int]*comments.Comment)
		return
	}

	filePath := m.file.NewPath
	if filePath == "" {
		filePath = m.file.OldPath
	}

	loadedComments, err := m.commentStorage.LoadComments(filePath)
	if err != nil {
		logger.Error("Failed to load comments: %v", err)
		m.fileComments = make(map[int]*comments.Comment)
	} else {
		m.fileComments = loadedComments
		logger.Info("Loaded %d comments for %s", len(loadedComments), filePath)
	}
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

	// Size the comment input
	m.commentInput.SetSize(width, height)

	// Experiment: Don't repaint on resize
}

// SetHighlightingEnabled enables or disables syntax highlighting
func (m *DiffViewModel) SetHighlightingEnabled(enabled bool) {
	m.highlightingEnabled = enabled
	// Clear cache to force re-render with new setting
	m.cachedContent = ""
	m.cachedFile = nil
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

	// Reset line number mapping
	m.lineToFileLineNum = make(map[int]int)

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

			// Track mapping between display line and file line number
			fileLineNum := line.NewNum
			if line.Type == ctypes.LineDeleted {
				fileLineNum = line.OldNum
			}
			m.lineToFileLineNum[lineNum] = fileLineNum

			// Check if this line has a comment
			hasComment := false
			if _, exists := m.fileComments[fileLineNum]; exists {
				hasComment = true
			}

			lineStr := m.renderLine(line, highlighted, lineNum, hasComment)
			b.WriteString(lineStr)
			// Add to navigable lines (only actual diff lines, not headers)
			navigableLines = append(navigableLines, lineNum)
			lineNum++
			b.WriteString("\n")
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

// renderLine renders a single diff line with pre-highlighted content
func (m *DiffViewModel) renderLine(line *ctypes.Line, highlightedContent string, currentLineNum int, hasComment bool) string {
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

	// Add comment indicator if comment exists
	commentIndicator := " "
	if hasComment {
		commentIndicator = "💬"
	}

	prefix := fmt.Sprintf("%4d %s %s ", lineNum, indicator, commentIndicator)

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
