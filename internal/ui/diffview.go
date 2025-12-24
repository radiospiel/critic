package ui

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"git.15b.it/eno/critic/internal/highlight"
	"git.15b.it/eno/critic/internal/logger"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var bgCodeRegex = regexp.MustCompile(`\x1b\[4[0-9](;[0-9]+)*m`)

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

	filename := m.file.NewPath
	if m.file.IsDeleted {
		filename = m.file.OldPath
	}

	// Render file header
	header := fmt.Sprintf("📄 %s", filename)
	if m.file.IsNew {
		header += " (new)"
	} else if m.file.IsDeleted {
		header += " (deleted)"
	} else if m.file.IsRenamed {
		header = fmt.Sprintf("📄 %s → %s (renamed)", m.file.OldPath, m.file.NewPath)
	}

	b.WriteString(m.renderLineWithCursor(hunkHeaderStyle.Render(header), lineNum))
	lineNum++
	b.WriteString("\n")
	b.WriteString(m.renderLineWithCursor("", lineNum)) // Empty line
	lineNum++
	b.WriteString("\n")

	// Highlight entire file(s) once if enabled, then extract lines by line number
	var oldFileHighlighted map[int]string // oldLineNum -> highlighted content
	var newFileHighlighted map[int]string // newLineNum -> highlighted content

	if m.highlightingEnabled {
		hlStart := time.Now()
		oldFileHighlighted = m.highlightFullFile(m.file, filename, true)  // old version
		newFileHighlighted = m.highlightFullFile(m.file, filename, false) // new version
		m.highlightTime += time.Since(hlStart)
	}

	// Render each hunk
	for hunkIdx, hunk := range m.file.Hunks {
		// Render hunk header
		hunkHeader := fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
		if hunk.Header != "" {
			hunkHeader += " " + hunk.Header
		}
		b.WriteString(m.renderLineWithCursor(hunkHeaderStyle.Render(hunkHeader), lineNum))
		lineNum++
		b.WriteString("\n")

		// Render hunk lines
		for _, line := range hunk.Lines {
			var highlighted string
			if m.highlightingEnabled {
				// Look up highlighted content by line number
				switch line.Type {
				case ctypes.LineAdded:
					if hl, ok := newFileHighlighted[line.NewNum]; ok {
						highlighted = hl
					} else {
						highlighted = line.Content
					}
				case ctypes.LineDeleted:
					if hl, ok := oldFileHighlighted[line.OldNum]; ok {
						highlighted = hl
					} else {
						highlighted = line.Content
					}
				case ctypes.LineContext:
					// Context exists in both, prefer new
					if hl, ok := newFileHighlighted[line.NewNum]; ok {
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
			lineNum++
			b.WriteString("\n")
		}

		// Add spacing between hunks
		if hunkIdx < len(m.file.Hunks)-1 {
			b.WriteString(m.renderLineWithCursor("", lineNum))
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

// highlightFullFile gets the complete file from git and returns highlighted lines by line number
func (m *DiffViewModel) highlightFullFile(file *ctypes.FileDiff, filename string, useOldVersion bool) map[int]string {
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
		// Use HEAD as the old version (works for diffs to working directory)
		// For "last commit" mode, HEAD is the new version and HEAD~1 would be old,
		// but we'll use HEAD as a reasonable default
		fullContent, err = m.getFileContentFromGit(file.OldPath, "HEAD")
	} else {
		// Get new version (current working directory or HEAD depending on mode)
		if file.IsDeleted {
			// Deleted file has no new version
			return result
		}
		// First try working directory, then fall back to HEAD
		fullContent, err = m.getFileContentFromGit(file.NewPath, "")
		if err != nil {
			// Fallback to HEAD for committed changes
			fullContent, err = m.getFileContentFromGit(file.NewPath, "HEAD")
		}
	}

	if err != nil || fullContent == "" {
		// Fallback to hunk-based reconstruction if git fails
		return m.highlightFromHunks(file, filename, useOldVersion)
	}

	// Highlight the complete file
	highlighted, err := m.highlighter.Highlight(fullContent, filename)
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

// getFileContentFromGit retrieves file content from git at a specific revision
// If revision is empty, reads from working directory
func (m *DiffViewModel) getFileContentFromGit(path string, revision string) (string, error) {
	var cmd *exec.Cmd

	if revision == "" {
		// Read from working directory
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	// Read from git at specific revision
	cmd = exec.Command("git", "show", revision+":"+path)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
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

	// Expand tabs to spaces for consistent rendering across all line types
	fullLine = expandTabsInANSI(fullLine)

	// Apply diff line styling with background that spans full width
	var styled string
	switch line.Type {
	case ctypes.LineAdded:
		styled = m.applyLineBackground(fullLine, "\x1b[48;2;26;58;26m") // Dark greenish
	case ctypes.LineDeleted:
		styled = m.applyLineBackground(fullLine, "\x1b[48;2;58;26;26m") // Dark reddish
	case ctypes.LineContext:
		styled = contextLineStyle.Render(fullLine)
	default:
		styled = fullLine
	}

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

// applyLineBackground wraps a line with background color spanning full width
func (m *DiffViewModel) applyLineBackground(line string, bgColor string) string {
	// Strip any background ANSI codes from syntax highlighting
	cleaned := stripBackgroundCodes(line)

	// Note: tabs are already expanded in renderLine before this function is called

	// Calculate visible width (accounting for ANSI codes and expanded tabs)
	visibleWidth := lipgloss.Width(cleaned)

	// Replace full resets with foreground-only resets (preserves background)
	// \x1b[0m resets everything, \x1b[39m resets only foreground
	cleaned = strings.ReplaceAll(cleaned, "\x1b[0m", "\x1b[39m")
	cleaned = strings.ReplaceAll(cleaned, "\x1b[m", "\x1b[39m")

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
	return bgColor + processed + "\x1b[0m"
}

// truncateANSI truncates a string with ANSI codes to a specific visible width
func truncateANSI(s string, maxWidth int) string {
	// Use lipgloss to truncate while preserving ANSI codes
	return lipgloss.NewStyle().MaxWidth(maxWidth).Render(s)
}

// stripBackgroundCodes removes background color ANSI codes from a string
func stripBackgroundCodes(s string) string {
	// Remove all background color codes (40-49 are background colors)
	return bgCodeRegex.ReplaceAllString(s, "")
}

// expandTabsInANSI expands tabs to spaces while preserving ANSI codes
func expandTabsInANSI(s string) string {
	const tabWidth = 4
	var result strings.Builder
	col := 0 // Current column position (visible characters)
	inANSI := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		// Track ANSI escape sequences (don't count toward column position)
		if ch == '\x1b' {
			inANSI = true
			result.WriteByte(ch)
			continue
		}

		if inANSI {
			result.WriteByte(ch)
			if ch == 'm' {
				inANSI = false
			}
			continue
		}

		// Expand tabs
		if ch == '\t' {
			// Calculate spaces needed to reach next tab stop
			spacesToAdd := tabWidth - (col % tabWidth)
			result.WriteString(strings.Repeat(" ", spacesToAdd))
			col += spacesToAdd
		} else {
			result.WriteByte(ch)
			col++
		}
	}

	return result.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
