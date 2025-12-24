package ui

import (
	"fmt"
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
			m.viewport, cmd = m.viewport.Update(msg)
		}

	case diffRenderedMsg:
		// Only apply if still viewing the same file
		if msg.file == m.file {
			m.cachedContent = msg.content
			if m.ready {
				m.viewport.SetContent(m.cachedContent)
				m.viewport.GotoTop()
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
		m.cachedContent = m.renderDiff()
		if m.ready {
			m.viewport.SetContent(m.cachedContent)
			m.viewport.GotoTop()
		}
	}
	return nil
}

// diffRenderedMsg is sent when async rendering completes
type diffRenderedMsg struct {
	file    *ctypes.FileDiff
	content string
}

// renderDiffAsync renders the diff in a background goroutine
func (m *DiffViewModel) renderDiffAsync(file *ctypes.FileDiff) tea.Cmd {
	return func() tea.Msg {
		// Render with highlighting in background
		content := m.renderDiff()
		return diffRenderedMsg{
			file:    file,
			content: content,
		}
	}
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
func (m *DiffViewModel) renderDiff() string {
	startTime := time.Now()
	m.highlightTime = 0 // Reset accumulated highlight time

	if m.file == nil {
		return ""
	}

	var b strings.Builder

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

	b.WriteString(hunkHeaderStyle.Render(header))
	b.WriteString("\n\n")

	// Render each hunk
	for hunkIdx, hunk := range m.file.Hunks {
		// Render hunk header
		hunkHeader := fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
		if hunk.Header != "" {
			hunkHeader += " " + hunk.Header
		}
		b.WriteString(hunkHeaderStyle.Render(hunkHeader))
		b.WriteString("\n")

		// Render hunk lines
		for _, line := range hunk.Lines {
			lineStr := m.renderLine(line, filename)
			b.WriteString(lineStr)
			b.WriteString("\n")
		}

		// Add spacing between hunks
		if hunkIdx < len(m.file.Hunks)-1 {
			b.WriteString("\n")
		}
	}

	result := b.String()
	totalTime := time.Since(startTime)
	renderTime := totalTime - m.highlightTime
	logger.Info("renderDiff: total=%v, highlight=%v, render=%v, lines=%d",
		totalTime, m.highlightTime, renderTime, m.countLines())

	return result
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

// renderLine renders a single diff line with syntax highlighting
func (m *DiffViewModel) renderLine(line *ctypes.Line, filename string) string {
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

	// Apply syntax highlighting if enabled
	var content string
	if m.highlightingEnabled {
		hlStart := time.Now()
		content = m.highlighter.HighlightLine(line.Content, filename)
		m.highlightTime += time.Since(hlStart)
	} else {
		content = line.Content
	}

	// Combine prefix with content
	fullLine := prefix + content

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

	return styled
}

// applyLineBackground wraps a line with background color spanning full width
func (m *DiffViewModel) applyLineBackground(line string, bgColor string) string {
	// Strip any background ANSI codes from syntax highlighting
	cleaned := stripBackgroundCodes(line)

	// Expand tabs to spaces (4-space tabs) - must do this for both width calc and rendering
	cleaned = expandTabsInANSI(cleaned)

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
