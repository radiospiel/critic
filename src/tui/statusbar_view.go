package tui

import (
	"fmt"
	"strings"
	"time"

	"github.org/radiospiel/critic/teapot"
	"github.com/charmbracelet/lipgloss"
)

// StatusBarView is a widget that displays the application status bar.
// Layout: [B]ase; [F]ilter; ? help -- diff stats: +/-/* NN/NN/NN -- clock
type StatusBarView struct {
	teapot.BaseView
	cellStyle lipgloss.Style // Style for individual cells (no padding)

	// Content sections
	base      string
	filter    string
	help      string
	diffStats string
	clock     string
}

// NewStatusBarView creates a new status bar widget.
func NewStatusBarView() *StatusBarView {
	// Use a cell style without padding for buffer rendering
	cellStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#6B95D8")).
		Foreground(lipgloss.Color("#000000"))

	sb := &StatusBarView{
		BaseView: teapot.NewBaseView(),
		cellStyle:  cellStyle,
		help:       "[?] help",
	}
	sb.SetFocusable(false)
	sb.SetConstraints(teapot.DefaultConstraints().WithMinSize(1, 1).WithPreferredSize(0, 1))
	return sb
}

// SetBase sets the base section text.
func (s *StatusBarView) SetBase(base string) {
	if base != "" {
		s.base = fmt.Sprintf("[B]ase: %s → HEAD", base)
	} else {
		s.base = ""
	}
	s.Repaint()
}

// SetFilter sets the filter section text.
func (s *StatusBarView) SetFilter(filter string) {
	s.filter = fmt.Sprintf("[F]ilter: %s", filter)
	s.Repaint()
}

// SetDiffStats sets the diff statistics section.
func (s *StatusBarView) SetDiffStats(added, deleted, changed int) {
	var parts []string
	if added > 0 {
		parts = append(parts, fmt.Sprintf("+%d", added))
	}
	if changed > 0 {
		parts = append(parts, fmt.Sprintf("~%d", changed))
	}
	if deleted > 0 {
		parts = append(parts, fmt.Sprintf("-%d", deleted))
	}
	if len(parts) > 0 {
		s.diffStats = strings.Join(parts, " ")
	} else {
		s.diffStats = ""
	}
	s.Repaint()
}

// ClearDiffStats clears the diff statistics.
func (s *StatusBarView) ClearDiffStats() {
	s.diffStats = ""
	s.Repaint()
}

// HandleTick implements teapot.TickHandler.
// Updates the clock and repaints only if the time changed.
func (s *StatusBarView) HandleTick() {
	newClock := time.Now().UTC().Format("15:04:05")
	if newClock != s.clock {
		s.clock = newClock
		s.Repaint()
	}
}

// Render renders the status bar to the buffer.
func (s *StatusBarView) Render(buf *teapot.SubBuffer) {
	width := buf.Width()
	height := buf.Height()

	// Fill background
	bgRow := strings.Repeat(" ", width)
	for y := 0; y < height; y++ {
		buf.SetString(0, y, bgRow, s.cellStyle)
	}

	// Build left section: [B]ase • [F]ilter • ? help
	var leftParts []string
	if s.base != "" {
		leftParts = append(leftParts, s.base)
	}
	if s.filter != "" {
		leftParts = append(leftParts, s.filter)
	}
	if s.help != "" {
		leftParts = append(leftParts, s.help)
	}

	leftText := ""
	for i, part := range leftParts {
		if i > 0 {
			leftText += " • "
		}
		leftText += part
	}

	// Add diff stats after help section
	if s.diffStats != "" {
		leftText += " -- " + s.diffStats
	}

	// Right section: clock
	rightText := s.clock
	rightLen := len(rightText)

	// Calculate available width for left content
	// Leave space for clock + 2 spaces padding
	availableForLeft := width - rightLen - 2
	if availableForLeft < 0 {
		availableForLeft = 0
	}

	// Render left section (truncate if needed)
	if len(leftText) > availableForLeft && availableForLeft > 3 {
		leftText = leftText[:availableForLeft-3] + "..."
	} else if len(leftText) > availableForLeft {
		leftText = leftText[:availableForLeft]
	}
	if len(leftText) > 0 {
		buf.SetStringTruncated(0, 0, leftText, len(leftText), s.cellStyle)
	}

	// Render clock at right edge
	if rightLen > 0 {
		clockX := width - rightLen
		if clockX < 0 {
			clockX = 0
		}
		buf.SetStringTruncated(clockX, 0, rightText, rightLen, s.cellStyle)
	}
}
