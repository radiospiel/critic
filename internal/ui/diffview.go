package ui

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/highlight"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiffViewModel represents the diff viewer pane
type DiffViewModel struct {
	file          *ctypes.FileDiff
	viewport      viewport.Model
	width         int
	height        int
	ready         bool
	highlighter   *highlight.Highlighter
	cachedContent string
	cachedFile    *ctypes.FileDiff
}

// NewDiffViewModel creates a new diff viewer model
func NewDiffViewModel() DiffViewModel {
	return DiffViewModel{
		highlighter: highlight.NewHighlighter(),
	}
}

// Init initializes the diff view model
func (m DiffViewModel) Init() tea.Cmd {
	return nil
}

// Update updates the diff view model
func (m DiffViewModel) Update(msg tea.Msg) (DiffViewModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.ready {
			m.viewport, cmd = m.viewport.Update(msg)
		}
	}

	return m, cmd
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

	// Use cached content if available and file hasn't changed
	var content string
	if m.cachedFile == m.file && m.cachedContent != "" {
		content = m.cachedContent
	} else {
		content = m.renderDiff()
	}

	if m.ready {
		m.viewport.SetContent(content)
		return m.viewport.View()
	}

	return content
}

// SetFile sets the current file to display
func (m *DiffViewModel) SetFile(file *ctypes.FileDiff) {
	m.file = file

	// Pre-render and cache the diff content
	if file != nil && (m.cachedFile != file) {
		m.cachedContent = m.renderDiff()
		m.cachedFile = file
	}

	if m.ready && file != nil {
		m.viewport.GotoTop()
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

// renderDiff renders the diff content with syntax highlighting
func (m *DiffViewModel) renderDiff() string {
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

	return b.String()
}

// renderLine renders a single diff line with syntax highlighting
func (m *DiffViewModel) renderLine(line *ctypes.Line, filename string) string {
	prefix := line.Type.String()

	// Highlight the content
	highlighted := m.highlighter.HighlightLine(line.Content, filename)

	// Apply diff line styling
	var styled string
	switch line.Type {
	case ctypes.LineAdded:
		styled = addedLineStyle.Render(prefix + highlighted)
	case ctypes.LineDeleted:
		styled = deletedLineStyle.Render(prefix + highlighted)
	case ctypes.LineContext:
		styled = contextLineStyle.Render(prefix + highlighted)
	default:
		styled = prefix + highlighted
	}

	return styled
}
