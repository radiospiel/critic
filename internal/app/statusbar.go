package app

import (
	"fmt"
	"strings"
	"time"

	ctypes "git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/internal/tui"
)

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	var parts []string

	// Show current base
	if len(m.bases) > 0 {
		base := m.bases[m.currentBase]
		parts = append(parts, fmt.Sprintf("[B]ase: %s → HEAD", base))
	}

	// Show filter mode (always visible in status bar)
	parts = append(parts, fmt.Sprintf("[f]ilter: %s", m.filterMode.String()))

	// Show file count and line stats
	if m.diff != nil {
		filteredCount, totalCount := m.fileList.GetFilterInfo()
		if m.filterMode == FilterModeNone {
			parts = append(parts, fmt.Sprintf("Files: %d", len(m.diff.Files)))
		} else {
			parts = append(parts, fmt.Sprintf("Files: %d/%d", filteredCount, totalCount))
		}

		// Show line statistics
		stats := computeDiffStats(m.diff)
		parts = append(parts, fmt.Sprintf("+%d -%d ~%d", stats.Added, stats.Deleted, stats.Moved))
	}

	// Show help hint
	parts = append(parts, "[?] help • [q] quit")

	leftStatus := strings.Join(parts, " • ")

	// Add UTC clock on the right side
	clock := time.Now().UTC().Format("15:04:05")

	// Calculate available width for left status (leave room for clock + padding)
	clockWidth := len(clock) + 2 // clock + 2 spaces padding
	availableWidth := m.width - clockWidth - 4
	if availableWidth > 3 && len(leftStatus) > availableWidth {
		leftStatus = leftStatus[:availableWidth-3] + "..."
	}

	// Pad left status to push clock to the right
	padding := m.width - len(leftStatus) - len(clock) - 2 // -2 for style padding
	if padding < 1 {
		padding = 1
	}
	status := leftStatus + strings.Repeat(" ", padding) + clock

	return tui.GetStatusStyle().
		Width(m.width).
		MaxWidth(m.width).
		Inline(true).
		Render(status)
}

// diffStats holds statistics about a diff
type diffStats struct {
	Added   int
	Deleted int
	Moved   int
}

// computeDiffStats computes line statistics for a diff
func computeDiffStats(diff *ctypes.Diff) diffStats {
	var stats diffStats
	if diff == nil {
		return stats
	}

	// First pass: count all added and deleted lines, track content for move detection
	addedLines := make(map[string]int)   // content -> count
	deletedLines := make(map[string]int) // content -> count

	for _, file := range diff.Files {
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				content := line.Content
				switch line.Type {
				case ctypes.LineAdded:
					stats.Added++
					addedLines[content]++
				case ctypes.LineDeleted:
					stats.Deleted++
					deletedLines[content]++
				}
			}
		}
	}

	// Detect moved lines: content that appears in both added and deleted
	for content, deletedCount := range deletedLines {
		if addedCount, ok := addedLines[content]; ok {
			// Count the minimum as moved (the rest are true adds/deletes)
			moved := deletedCount
			if addedCount < moved {
				moved = addedCount
			}
			stats.Moved += moved
		}
	}

	// Adjust added/deleted to exclude moved lines
	stats.Added -= stats.Moved
	stats.Deleted -= stats.Moved

	return stats
}
