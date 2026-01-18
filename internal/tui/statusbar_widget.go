package tui

import (
	"fmt"
	"strings"
	"time"

	"git.15b.it/eno/critic/teapot"
	"github.com/charmbracelet/lipgloss"
)

// StatusBarWidget is a widget that displays the application status bar.
// Layout: [B]ase; [F]ilter; ? help -- diff stats: +/-/* NN/NN/NN -- clock
type StatusBarWidget struct {
	teapot.BaseWidget
	cellStyle lipgloss.Style // Style for individual cells (no padding)

	// Content sections
	base      string
	filter    string
	help      string
	diffStats string
	clock     string
}

// NewStatusBarWidget creates a new status bar widget.
func NewStatusBarWidget() *StatusBarWidget {
	// Use a cell style without padding for buffer rendering
	cellStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#6B95D8")).
		Foreground(lipgloss.Color("#000000"))

	sb := &StatusBarWidget{
		BaseWidget: teapot.NewBaseWidget(),
		cellStyle:  cellStyle,
		help:       "[?] help",
	}
	sb.SetFocusable(false)
	sb.SetConstraints(teapot.DefaultConstraints().WithMinSize(1, 1).WithPreferredSize(0, 1))
	return sb
}

// SetBase sets the base section text.
func (s *StatusBarWidget) SetBase(base string) {
	if base != "" {
		s.base = fmt.Sprintf("[B]ase: %s → HEAD", base)
	} else {
		s.base = ""
	}
	s.Invalidate()
}

// SetFilter sets the filter section text.
func (s *StatusBarWidget) SetFilter(filter string) {
	s.filter = fmt.Sprintf("[F]ilter: %s", filter)
	s.Invalidate()
}

// SetDiffStats sets the diff statistics section.
func (s *StatusBarWidget) SetDiffStats(added, deleted, moved int) {
	s.diffStats = fmt.Sprintf("+%d/-%d/~%d", added, deleted, moved)
	s.Invalidate()
}

// ClearDiffStats clears the diff statistics.
func (s *StatusBarWidget) ClearDiffStats() {
	s.diffStats = ""
	s.Invalidate()
}

// HandleTick implements teapot.TickHandler.
// Updates the clock and repaints only if the time changed.
func (s *StatusBarWidget) HandleTick() {
	newClock := time.Now().UTC().Format("15:04:05")
	if newClock != s.clock {
		s.clock = newClock
		s.Invalidate()
	}
}

// Render renders the status bar to the buffer.
func (s *StatusBarWidget) Render(buf *teapot.SubBuffer) {
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
