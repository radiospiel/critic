package tui

import (
	"strings"
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
	"git.15b.it/eno/critic/teapot"
)

func TestStatusBarView_Render(t *testing.T) {
	sb := NewStatusBarView()
	sb.SetBase("origin/master")
	sb.SetFilter("All")
	sb.SetDiffStats(10, 5, 2)
	sb.HandleTick()

	// Render to a buffer (use 100 width to fit all content)
	width := 100
	buf := teapot.NewBuffer(width, 1)
	subBuf := teapot.NewSubBuffer(buf, teapot.Rect{X: 0, Y: 0, Width: width, Height: 1})
	sb.Render(subBuf)

	// Extract text content from buffer (runes only, no ANSI codes)
	var runes []rune
	for x := 0; x < width; x++ {
		cell := buf.GetCell(x, 0)
		runes = append(runes, cell.Rune)
	}
	content := string(runes)

	// Verify no double spaces (except for the gap before clock)
	// The content should have single spaces, not double
	assert.False(t, strings.Contains(content, "  B  "), "should not have spaces around each character")

	// Verify expected content is present
	assert.Contains(t, content, "[B]ase:", "should contain base section")
	assert.Contains(t, content, "origin/master", "should contain base ref")
	assert.Contains(t, content, "[F]ilter:", "should contain filter section")
	assert.Contains(t, content, "All", "should contain filter value")
	assert.Contains(t, content, "[?] help", "should contain help section")
	assert.Contains(t, content, "+10/-5/~2", "should contain diff stats")
}

func TestStatusBarView_ClockAtRight(t *testing.T) {
	sb := NewStatusBarView()
	sb.SetBase("main")
	sb.SetFilter("All")
	sb.HandleTick()

	// Render to a buffer
	width := 60
	buf := teapot.NewBuffer(width, 1)
	subBuf := teapot.NewSubBuffer(buf, teapot.Rect{X: 0, Y: 0, Width: width, Height: 1})
	sb.Render(subBuf)

	// Extract text content from buffer
	var runes []rune
	for x := 0; x < width; x++ {
		cell := buf.GetCell(x, 0)
		runes = append(runes, cell.Rune)
	}
	content := string(runes)

	// The clock should be at the right edge (last 8 characters: HH:MM:SS)
	rightPart := strings.TrimRight(content, " ")
	assert.True(t, len(rightPart) >= 8, "content should have at least 8 chars for clock")

	// Check clock format (HH:MM:SS) at the end
	clockPart := rightPart[len(rightPart)-8:]
	assert.Equals(t, clockPart[2], byte(':'), "clock should have : at position 2")
	assert.Equals(t, clockPart[5], byte(':'), "clock should have : at position 5")
}

func TestStatusBarView_NoDoubleSpaces(t *testing.T) {
	sb := NewStatusBarView()
	sb.SetBase("origin/master")
	sb.SetFilter("With Comments")
	sb.SetDiffStats(100, 50, 25)
	sb.HandleTick()

	// Render to buffer and check cell by cell
	width := 100
	buf := teapot.NewBuffer(width, 1)
	subBuf := teapot.NewSubBuffer(buf, teapot.Rect{X: 0, Y: 0, Width: width, Height: 1})
	sb.Render(subBuf)

	// Check that we don't have the pattern "char space char space char"
	// which would indicate each char is being padded
	var runes []rune
	for x := 0; x < width; x++ {
		cell := buf.GetCell(x, 0)
		runes = append(runes, cell.Rune)
	}

	// Look for the pattern that indicates padding issue: " B " " a " " s " " e "
	content := string(runes)
	hasDoublePadding := strings.Contains(content, " B ") &&
		strings.Contains(content, " a ") &&
		strings.Contains(content, " s ") &&
		strings.Contains(content, " e ")
	assert.False(t, hasDoublePadding, "should not have padding around each character: %s", content)
}

func TestStatusBarView_EmptySections(t *testing.T) {
	sb := NewStatusBarView()
	// Don't set any sections except clock
	sb.HandleTick()

	width := 40
	buf := teapot.NewBuffer(width, 1)
	subBuf := teapot.NewSubBuffer(buf, teapot.Rect{X: 0, Y: 0, Width: width, Height: 1})
	sb.Render(subBuf)

	// Should still render without panic
	var runes []rune
	for x := 0; x < width; x++ {
		cell := buf.GetCell(x, 0)
		runes = append(runes, cell.Rune)
	}
	content := string(runes)

	// Should contain help section (always present)
	assert.Contains(t, content, "[?] help", "should contain help section even with empty base/filter")
}

func TestStatusBarView_Truncation(t *testing.T) {
	sb := NewStatusBarView()
	sb.SetBase("very-long-branch-name-that-will-need-truncation")
	sb.SetFilter("With Comments")
	sb.SetDiffStats(100, 50, 25)
	sb.HandleTick()

	// Use narrow width to force truncation
	width := 60
	buf := teapot.NewBuffer(width, 1)
	subBuf := teapot.NewSubBuffer(buf, teapot.Rect{X: 0, Y: 0, Width: width, Height: 1})
	sb.Render(subBuf)

	var runes []rune
	for x := 0; x < width; x++ {
		cell := buf.GetCell(x, 0)
		runes = append(runes, cell.Rune)
	}
	content := string(runes)

	// Should contain truncation indicator
	assert.Contains(t, content, "...", "should contain truncation indicator when content is too long")

	// Clock should still be at the right edge
	rightPart := strings.TrimRight(content, " ")
	clockPart := rightPart[len(rightPart)-8:]
	assert.Equals(t, clockPart[2], byte(':'), "clock should be at right edge with : at position 2")
}
