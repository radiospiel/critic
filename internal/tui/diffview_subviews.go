package tui

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/session"
	"git.15b.it/eno/critic/pkg/critic"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/teapot"
	"github.com/charmbracelet/lipgloss"
)

// HunkView renders a complete hunk: header + lines + inline comments.
// This is the primary building block of the diff view.
type HunkView struct {
	teapot.BaseView
	hunk            *ctypes.Hunk
	conversationMap map[int]*critic.Conversation
	highlightedOld  map[int]string // Pre-highlighted content for deleted lines
	highlightedNew  map[int]string // Pre-highlighted content for added lines
	highlightedCtx  map[int]string // Pre-highlighted content for context lines
	selectedRow     int            // Which row within this hunk is selected (-1 = none)
	startRow        int            // Global row number where this hunk starts (for selection mapping)
}

// NewHunkView creates a new hunk widget.
func NewHunkView(
	hunk *ctypes.Hunk,
	conversationMap map[int]*critic.Conversation,
	highlightedOld, highlightedNew, highlightedCtx map[int]string,
) *HunkView {
	w := &HunkView{
		BaseView:        teapot.NewBaseView(),
		hunk:            hunk,
		conversationMap: conversationMap,
		highlightedOld:  highlightedOld,
		highlightedNew:  highlightedNew,
		highlightedCtx:  highlightedCtx,
		selectedRow:     -1,
	}
	w.SetFocusable(false)

	// Calculate height: 1 (header) + lines + comments
	height := w.calculateHeight()
	w.SetConstraints(teapot.DefaultConstraints().WithMinSize(1, height).WithPreferredSize(0, height))
	return w
}

// calculateHeight returns the total height of this hunk.
func (w *HunkView) calculateHeight() int {
	height := 1 // Hunk header
	for _, line := range w.hunk.Lines {
		height++ // Line
		if line.NewNum > 0 {
			if conv, exists := w.conversationMap[line.NewNum]; exists {
				height += calculateCommentHeight(conv)
			}
		}
	}
	return height
}

// calculateCommentHeight returns the height needed for a comment.
func calculateCommentHeight(conv *critic.Conversation) int {
	// separator + content lines + separator
	contentLines := 0
	for _, msg := range conv.Messages {
		contentLines += len(strings.Split(msg.Message, "\n"))
	}
	return 1 + contentLines + 1
}

// SetSelectedRow sets which row within this hunk is selected.
func (w *HunkView) SetSelectedRow(row int) {
	w.selectedRow = row
}

// SetStartRow sets the global row number where this hunk starts.
func (w *HunkView) SetStartRow(row int) {
	w.startRow = row
}

// Hunk returns the underlying hunk.
func (w *HunkView) Hunk() *ctypes.Hunk {
	return w.hunk
}

// Render renders the hunk to the buffer.
func (w *HunkView) Render(buf *teapot.SubBuffer) {
	w.RenderWithYOffset(buf, 0)
}

// RenderWithYOffset renders the hunk with an optional Y offset for partial visibility.
// ySkip specifies how many lines at the top to skip (used when the hunk is scrolled
// and its top portion is above the visible area).
func (w *HunkView) RenderWithYOffset(buf *teapot.SubBuffer, ySkip int) {
	width := buf.Width()
	if width <= 0 {
		return
	}

	// y tracks our position in the hunk's logical coordinate space
	// renderY tracks where we actually render in the buffer
	y := 0
	renderY := y - ySkip

	// Render hunk header
	header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", w.hunk.OldStart, w.hunk.OldLines, w.hunk.NewStart, w.hunk.NewLines)
	if w.hunk.Header != "" {
		header += " " + w.hunk.Header
	}
	if renderY >= 0 {
		w.renderHeaderLine(buf, renderY, header, width)
	}
	y++
	renderY = y - ySkip

	// Render each line and its comment (if any)
	// Note: Selection highlighting is done via buffer overlay, not here
	for _, line := range w.hunk.Lines {
		// Get highlighted content
		highlighted := w.getHighlightedContent(line)

		// Render the diff line
		if renderY >= 0 {
			w.renderDiffLine(buf, renderY, line, highlighted, width)
		}
		y++
		renderY = y - ySkip

		// Render comment if exists
		if line.NewNum > 0 {
			if conv, exists := w.conversationMap[line.NewNum]; exists {
				commentHeight := calculateCommentHeight(conv)
				// Check if any row in comment is selected (for hotkey display)
				commentSelected := w.selectedRow >= y && w.selectedRow < y+commentHeight
				if renderY >= 0 || renderY+commentHeight > 0 {
					w.renderCommentWithYOffset(buf, renderY, conv, width, commentHeight, commentSelected, ySkip-y)
				}
				y += commentHeight
				renderY = y - ySkip
			}
		}
	}
}

// getHighlightedContent returns the highlighted content for a line.
func (w *HunkView) getHighlightedContent(line *ctypes.Line) string {
	switch line.Type {
	case ctypes.LineAdded:
		if hl, ok := w.highlightedNew[line.NewNum]; ok {
			return hl
		}
	case ctypes.LineDeleted:
		if hl, ok := w.highlightedOld[line.OldNum]; ok {
			return hl
		}
	case ctypes.LineContext:
		if hl, ok := w.highlightedCtx[line.NewNum]; ok {
			return hl
		}
	}
	return line.Content
}

// renderHeaderLine renders the hunk header line.
func (w *HunkView) renderHeaderLine(buf *teapot.SubBuffer, y int, header string, width int) {
	if y >= buf.Height() {
		return
	}
	// Build cells for the header line, padding with spaces
	cells := make([]teapot.Cell, width)
	runes := []rune(header)
	for x := 0; x < width; x++ {
		if x < len(runes) {
			cells[x] = teapot.Cell{Rune: runes[x], Style: hunkHeaderStyle}
		} else {
			cells[x] = teapot.Cell{Rune: ' ', Style: hunkHeaderStyle}
		}
	}
	buf.SetCells(0, y, cells)
}

// renderDiffLine renders a single diff line (selection highlighting is done via overlay).
func (w *HunkView) renderDiffLine(buf *teapot.SubBuffer, y int, line *ctypes.Line, highlighted string, width int) {
	if y >= buf.Height() {
		return
	}

	// Build line number prefix
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

	// Get background color
	var bgColor lipgloss.Color
	switch line.Type {
	case ctypes.LineAdded:
		bgColor = lipgloss.Color("22") // Dark green
	case ctypes.LineDeleted:
		bgColor = lipgloss.Color("52") // Dark red
	default:
		bgColor = lipgloss.Color("0") // Black
	}

	style := lipgloss.NewStyle().Background(bgColor)

	// Render prefix + content
	content := prefix + highlighted
	parsedCells := teapot.ParseANSILine(content)

	// Build row of cells, padding with spaces
	rowCells := make([]teapot.Cell, width)
	for x := 0; x < width; x++ {
		if x < len(parsedCells) {
			rowCells[x] = parsedCells[x]
			rowCells[x].Style = rowCells[x].Style.Background(bgColor)
		} else {
			rowCells[x] = teapot.Cell{Rune: ' ', Style: style}
		}
	}
	buf.SetCells(0, y, rowCells)
}

// renderComment renders an inline comment/conversation.
func (w *HunkView) renderComment(buf *teapot.SubBuffer, startY int, conv *critic.Conversation, width, height int, selected bool) {
	if startY >= buf.Height() {
		return
	}

	// Styles
	lightBlueBg := lipgloss.Color("#6B95D8")
	blackFg := lipgloss.Color("0")
	grayFg := lipgloss.Color("240")

	contentStyle := lipgloss.NewStyle().Background(lightBlueBg).Foreground(blackFg)
	separatorStyle := lipgloss.NewStyle().Foreground(grayFg)

	y := startY

	// Top separator with animation
	if y < buf.Height() {
		// Render the animation frame (first 12 chars)
		animFrame := GetSeparatorFrame()
		animCells := teapot.ParseANSILine(animFrame)
		if len(animCells) > width {
			animCells = animCells[:width]
		}
		buf.SetCells(0, y, animCells)

		// Render the rest of the separator (space + dashes)
		if width > 12 {
			buf.SetString(12, y, " ", separatorStyle)
		}
		if width > 13 {
			buf.SetString(13, y, strings.Repeat("-", width-13), separatorStyle)
		}
		y++
	}

	// Build content lines
	var contentLines []string
	for i, msg := range conv.Messages {
		prefix := "You"
		if msg.Author == critic.AuthorAI {
			prefix = "AI"
		}

		if i == 0 && msg.Author == critic.AuthorHuman {
			msgLines := strings.Split(msg.Message, "\n")
			for _, line := range msgLines {
				contentLines = append(contentLines, renderMarkdown(line))
			}
		} else {
			replyLines := strings.Split(msg.Message, "\n")
			for j, line := range replyLines {
				if j == 0 {
					contentLines = append(contentLines, fmt.Sprintf("%s: %s", prefix, renderMarkdown(line)))
				} else {
					indent := strings.Repeat(" ", len(prefix)+2)
					contentLines = append(contentLines, indent+renderMarkdown(line))
				}
			}
		}
	}

	// Prepend resolved status
	if conv.Status == critic.StatusResolved && len(contentLines) > 0 {
		contentLines[0] = "(Resolved) " + contentLines[0]
	}

	// Render content lines (selection highlighting is done via overlay)
	for _, line := range contentLines {
		if y >= buf.Height() {
			break
		}

		content := " " + line
		parsedCells := teapot.ParseANSILine(content)

		// Build row of cells with padding
		rowCells := make([]teapot.Cell, width)
		for x := 0; x < width; x++ {
			if x < len(parsedCells) {
				rowCells[x] = parsedCells[x]
				rowCells[x].Style = rowCells[x].Style.Background(lightBlueBg).Foreground(blackFg)
			} else {
				rowCells[x] = teapot.Cell{Rune: ' ', Style: contentStyle}
			}
		}
		buf.SetCells(0, y, rowCells)
		y++
	}

	// Bottom separator with hotkeys if selected
	if y < buf.Height() {
		var separatorText string
		if selected {
			separatorText = "[R]esolve - [Enter] reply"
			if conv.Status == critic.StatusResolved {
				separatorText = "[R] unresolve - [Enter] reply"
			}
		}

		var bottomLine string
		if separatorText == "" {
			bottomLine = strings.Repeat("-", width)
		} else {
			textLen := len(separatorText)
			leftDashes := (width - textLen - 2) / 2
			rightDashes := width - textLen - 2 - leftDashes
			if leftDashes < 0 {
				leftDashes = 0
			}
			if rightDashes < 0 {
				rightDashes = 0
			}
			bottomLine = strings.Repeat("-", leftDashes) + " " + separatorText + " " + strings.Repeat("-", rightDashes)
		}

		// Pad to width if needed
		if len(bottomLine) < width {
			bottomLine += strings.Repeat("-", width-len(bottomLine))
		}
		buf.SetString(0, y, bottomLine, separatorStyle)
	}
}

// renderCommentWithYOffset renders an inline comment with a Y offset for partial visibility.
// ySkip specifies how many lines at the top of the comment to skip.
func (w *HunkView) renderCommentWithYOffset(buf *teapot.SubBuffer, startY int, conv *critic.Conversation, width, height int, selected bool, ySkip int) {
	if ySkip <= 0 {
		// No offset needed, use regular render
		w.renderComment(buf, startY, conv, width, height, selected)
		return
	}

	if startY >= buf.Height() {
		return
	}

	// Styles
	lightBlueBg := lipgloss.Color("#6B95D8")
	blackFg := lipgloss.Color("0")
	grayFg := lipgloss.Color("240")

	contentStyle := lipgloss.NewStyle().Background(lightBlueBg).Foreground(blackFg)
	separatorStyle := lipgloss.NewStyle().Foreground(grayFg)

	// Track logical Y position and render Y position
	logicalY := 0
	renderY := startY

	// Top separator with animation (skip if ySkip > 0)
	if logicalY >= ySkip && renderY >= 0 && renderY < buf.Height() {
		// Render the animation frame (first 12 chars)
		animFrame := GetSeparatorFrame()
		animCells := teapot.ParseANSILine(animFrame)
		if len(animCells) > width {
			animCells = animCells[:width]
		}
		buf.SetCells(0, renderY, animCells)

		// Render the rest of the separator (space + dashes)
		if width > 12 {
			buf.SetString(12, renderY, " ", separatorStyle)
		}
		if width > 13 {
			buf.SetString(13, renderY, strings.Repeat("-", width-13), separatorStyle)
		}
		renderY++
	} else if logicalY < ySkip {
		// Skip this line but still advance render position if we're past the skip
		if logicalY >= ySkip {
			renderY++
		}
	}
	logicalY++

	// Build content lines
	var contentLines []string
	for i, msg := range conv.Messages {
		prefix := "You"
		if msg.Author == critic.AuthorAI {
			prefix = "AI"
		}

		if i == 0 && msg.Author == critic.AuthorHuman {
			msgLines := strings.Split(msg.Message, "\n")
			for _, line := range msgLines {
				contentLines = append(contentLines, renderMarkdown(line))
			}
		} else {
			replyLines := strings.Split(msg.Message, "\n")
			for j, line := range replyLines {
				if j == 0 {
					contentLines = append(contentLines, fmt.Sprintf("%s: %s", prefix, renderMarkdown(line)))
				} else {
					indent := strings.Repeat(" ", len(prefix)+2)
					contentLines = append(contentLines, indent+renderMarkdown(line))
				}
			}
		}
	}

	// Prepend resolved status
	if conv.Status == critic.StatusResolved && len(contentLines) > 0 {
		contentLines[0] = "(Resolved) " + contentLines[0]
	}

	// Render content lines
	for _, line := range contentLines {
		if logicalY >= ySkip && renderY >= 0 && renderY < buf.Height() {
			content := " " + line
			parsedCells := teapot.ParseANSILine(content)

			// Build row of cells with padding
			rowCells := make([]teapot.Cell, width)
			for x := 0; x < width; x++ {
				if x < len(parsedCells) {
					rowCells[x] = parsedCells[x]
					rowCells[x].Style = rowCells[x].Style.Background(lightBlueBg).Foreground(blackFg)
				} else {
					rowCells[x] = teapot.Cell{Rune: ' ', Style: contentStyle}
				}
			}
			buf.SetCells(0, renderY, rowCells)
			renderY++
		} else if logicalY >= ySkip {
			renderY++
		}
		logicalY++
	}

	// Bottom separator with hotkeys if selected
	if logicalY >= ySkip && renderY >= 0 && renderY < buf.Height() {
		var separatorText string
		if selected {
			separatorText = "[R]esolve - [Enter] reply"
			if conv.Status == critic.StatusResolved {
				separatorText = "[R] unresolve - [Enter] reply"
			}
		}

		var bottomLine string
		if separatorText == "" {
			bottomLine = strings.Repeat("-", width)
		} else {
			textLen := len(separatorText)
			leftDashes := (width - textLen - 2) / 2
			rightDashes := width - textLen - 2 - leftDashes
			if leftDashes < 0 {
				leftDashes = 0
			}
			if rightDashes < 0 {
				rightDashes = 0
			}
			bottomLine = strings.Repeat("-", leftDashes) + " " + separatorText + " " + strings.Repeat("-", rightDashes)
		}

		// Pad to width if needed
		if len(bottomLine) < width {
			bottomLine += strings.Repeat("-", width-len(bottomLine))
		}
		buf.SetString(0, renderY, bottomLine, separatorStyle)
	}
}

// FileHeaderView displays the file header with change statistics.
type FileHeaderView struct {
	teapot.BaseView
	file *ctypes.FileDiff
}

// NewFileHeaderView creates a new file header widget.
func NewFileHeaderView(file *ctypes.FileDiff) *FileHeaderView {
	w := &FileHeaderView{
		BaseView: teapot.NewBaseView(),
		file:     file,
	}
	w.SetFocusable(false)
	w.SetConstraints(teapot.DefaultConstraints().WithMinSize(1, 2).WithPreferredSize(0, 2))
	return w
}

// calculateStats returns added, deleted, and changed line counts.
// Changed lines are estimated as min(added, deleted) - representing modifications.
// Pure added = total added - changed, pure deleted = total deleted - changed.
func (w *FileHeaderView) calculateStats() (added, deleted, changed int) {
	if w.file == nil {
		return 0, 0, 0
	}
	var totalAdded, totalDeleted int
	for _, hunk := range w.file.Hunks {
		for _, line := range hunk.Lines {
			switch line.Type {
			case ctypes.LineAdded:
				totalAdded++
			case ctypes.LineDeleted:
				totalDeleted++
			}
		}
	}
	// Changed = lines that were modified (paired add+delete)
	changed = min(totalAdded, totalDeleted)
	added = totalAdded - changed
	deleted = totalDeleted - changed
	return added, deleted, changed
}

// Render renders the file header with stats.
func (w *FileHeaderView) Render(buf *teapot.SubBuffer) {
	width := buf.Width()
	if width <= 0 || w.file == nil {
		return
	}

	// Build file path display
	var filePath string
	if w.file.IsDeleted {
		filePath = w.file.OldPath + " (deleted)"
	} else if w.file.IsNew {
		filePath = w.file.NewPath + " (new)"
	} else if w.file.IsRenamed {
		filePath = w.file.OldPath + " → " + w.file.NewPath
	} else {
		filePath = w.file.NewPath
	}

	// Calculate stats
	added, deleted, changed := w.calculateStats()
	total := added + deleted + changed

	// Build stats string
	var statsStr string
	if total > 0 {
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
		statsStr = strings.Join(parts, " ")
	} else {
		statsStr = "no changes"
	}

	// Style similar to status bar - solid background
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#4A6FA5")). // Slightly darker blue than status bar
		Foreground(lipgloss.Color("#FFFFFF")). // White text
		Bold(true)

	statsStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#4A6FA5")).
		Foreground(lipgloss.Color("#CCCCCC")) // Slightly dimmer for stats

	// First line: file path on left, stats on right
	// Calculate spacing
	pathLen := len([]rune(filePath))
	statsLen := len(statsStr)
	padding := width - pathLen - statsLen - 2 // -2 for margins
	if padding < 1 {
		padding = 1
	}

	// Build first line with proper padding
	line1 := " " + filePath + strings.Repeat(" ", padding) + statsStr + " "
	if len([]rune(line1)) > width {
		// Truncate path if needed
		runes := []rune(line1)
		line1 = string(runes[:width])
	} else if len([]rune(line1)) < width {
		// Pad to full width
		line1 = line1 + strings.Repeat(" ", width-len([]rune(line1)))
	}

	// Render first line - path part
	cells := make([]teapot.Cell, width)
	runes := []rune(line1)
	pathEnd := 1 + pathLen // After leading space and path
	for x := 0; x < width; x++ {
		r := ' '
		if x < len(runes) {
			r = runes[x]
		}
		style := headerStyle
		if x >= pathEnd {
			style = statsStyle
		}
		cells[x] = teapot.Cell{Rune: r, Style: style}
	}
	buf.SetCells(0, 0, cells)

	// Second line: thin change line visualization
	if buf.Height() > 1 {
		w.renderChangeLine(buf, 1, width, added, deleted, changed)
	}
}

// renderChangeLine renders a thin line at the top of the cell (appearing as underline to row above).
func (w *FileHeaderView) renderChangeLine(buf *teapot.SubBuffer, y, width, added, deleted, changed int) {
	total := added + deleted + changed

	// Use UPPER ONE EIGHTH BLOCK (▔) - renders at top of cell, looks like underline
	const lineChar = '▔'

	if total == 0 {
		// Empty line
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
		buf.SetString(0, y, strings.Repeat(string(lineChar), width), emptyStyle)
		return
	}

	// Calculate widths for each segment
	addedWidth := (added * width) / total
	changedWidth := (changed * width) / total
	deletedWidth := width - addedWidth - changedWidth

	// Ensure at least 1 char for non-zero values
	if added > 0 && addedWidth == 0 {
		addedWidth = 1
		deletedWidth--
	}
	if changed > 0 && changedWidth == 0 {
		changedWidth = 1
		deletedWidth--
	}
	if deleted > 0 && deletedWidth == 0 {
		deletedWidth = 1
		if changedWidth > 1 {
			changedWidth--
		} else if addedWidth > 1 {
			addedWidth--
		}
	}
	if deletedWidth < 0 {
		deletedWidth = 0
	}

	// Styles for each segment
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50"))   // Green
	changedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFC107")) // Yellow/Amber
	deletedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F44336")) // Red

	// Build the line
	cells := make([]teapot.Cell, width)
	x := 0

	// Added segment (green)
	for i := 0; i < addedWidth && x < width; i++ {
		cells[x] = teapot.Cell{Rune: lineChar, Style: addedStyle}
		x++
	}

	// Changed segment (yellow)
	for i := 0; i < changedWidth && x < width; i++ {
		cells[x] = teapot.Cell{Rune: lineChar, Style: changedStyle}
		x++
	}

	// Deleted segment (red)
	for i := 0; i < deletedWidth && x < width; i++ {
		cells[x] = teapot.Cell{Rune: lineChar, Style: deletedStyle}
		x++
	}

	// Fill any remaining (shouldn't happen, but safety)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
	for x < width {
		cells[x] = teapot.Cell{Rune: lineChar, Style: emptyStyle}
		x++
	}

	buf.SetCells(0, y, cells)
}

// DiffContentView is the main widget that displays the entire diff view.
// It uses a ScrollView to handle scrolling, with a fixed file header.
type DiffContentView struct {
	teapot.BaseView
	scrollView      *teapot.ScrollView
	file            *ctypes.FileDiff
	hunks           []*HunkView
	conversationMap map[int]*critic.Conversation
	highlightedOld  map[int]string
	highlightedNew  map[int]string
	highlightedCtx  map[int]string
	filterMode      session.FilterMode
	selectedRow     int // Global row number of selected item
}

// NewDiffContentView creates a new diff view widget.
func NewDiffContentView() *DiffContentView {
	w := &DiffContentView{
		BaseView:   teapot.NewBaseView(),
		scrollView: teapot.NewScrollView(),
	}
	w.scrollView.SetParent(w)
	w.SetFocusable(true)
	return w
}

// SetFile sets the file to display and rebuilds the hunk widgets.
func (w *DiffContentView) SetFile(
	file *ctypes.FileDiff,
	conversationMap map[int]*critic.Conversation,
	highlightedOld, highlightedNew, highlightedCtx map[int]string,
) {
	w.file = file
	w.conversationMap = conversationMap
	w.highlightedOld = highlightedOld
	w.highlightedNew = highlightedNew
	w.highlightedCtx = highlightedCtx
	w.rebuildHunks()
	w.scrollView.ScrollToTop()
	w.selectedRow = 3 // First line after header (row 0-1) and first hunk header (row 2)
}

// SetFilterMode sets the filter mode.
func (w *DiffContentView) SetFilterMode(mode session.FilterMode) {
	w.filterMode = mode
}

// rebuildHunks creates HunkViews for each hunk and populates the ScrollView.
func (w *DiffContentView) rebuildHunks() {
	w.hunks = nil
	w.scrollView.ClearChildren()

	if w.file == nil {
		w.scrollView.SetHeaderView(nil)
		return
	}

	// Set file header as the fixed header
	w.scrollView.SetHeaderView(NewFileHeaderView(w.file))

	hunksToRender := w.filterHunks(w.file.Hunks)
	currentRow := 2 // After file header

	for _, hunk := range hunksToRender {
		hw := NewHunkView(hunk, w.conversationMap, w.highlightedOld, w.highlightedNew, w.highlightedCtx)
		hw.SetStartRow(currentRow)
		w.hunks = append(w.hunks, hw)
		w.scrollView.AddChild(hw)
		currentRow += hw.calculateHeight() + 1 // +1 for spacing
	}
}

// filterHunks filters hunks based on the current filter mode.
func (w *DiffContentView) filterHunks(hunks []*ctypes.Hunk) []*ctypes.Hunk {
	if w.filterMode == session.FilterModeNone {
		return hunks
	}

	var filtered []*ctypes.Hunk
	for _, hunk := range hunks {
		if w.hunkMatchesFilter(hunk) {
			filtered = append(filtered, hunk)
		}
	}
	return filtered
}

// hunkMatchesFilter checks if a hunk matches the current filter.
func (w *DiffContentView) hunkMatchesFilter(hunk *ctypes.Hunk) bool {
	for _, line := range hunk.Lines {
		if line.NewNum > 0 {
			if conv, exists := w.conversationMap[line.NewNum]; exists {
				switch w.filterMode {
				case session.FilterModeWithComments:
					return true
				case session.FilterModeWithUnresolved:
					if conv.Status != critic.StatusResolved {
						return true
					}
				}
			}
		}
	}
	return false
}

// GetSelectedRow returns the currently selected row.
func (w *DiffContentView) GetSelectedRow() int {
	return w.selectedRow
}

// SetSelectedRow sets the selected row.
func (w *DiffContentView) SetSelectedRow(row int) {
	w.selectedRow = row
}

// GetYOffset returns the current scroll offset.
func (w *DiffContentView) GetYOffset() int {
	return w.scrollView.ScrollOffset()
}

// SetBounds sets the bounds and propagates to the scroll view.
func (w *DiffContentView) SetBounds(bounds teapot.Rect) {
	w.BaseView.SetBounds(bounds)
	w.scrollView.SetBounds(bounds)
}

// GetFile returns the current file.
func (w *DiffContentView) GetFile() *ctypes.FileDiff {
	return w.file
}

// CalculateTotalHeight returns the total content height.
func (w *DiffContentView) CalculateTotalHeight() int {
	if w.file == nil {
		return 0
	}

	height := 2 // File header
	for i, hw := range w.hunks {
		height += hw.calculateHeight()
		if i < len(w.hunks)-1 {
			height++ // Spacing
		}
	}
	return height
}

// Render renders the entire diff view.
func (w *DiffContentView) Render(buf *teapot.SubBuffer) {
	if w.file == nil {
		return
	}

	// Adjust scroll offset to keep selected item visible
	w.ensureSelectedVisible()

	// Update selection state on each hunk before rendering
	currentRow := 2 // After file header
	for _, hw := range w.hunks {
		hunkHeight := hw.calculateHeight()

		// Calculate which row within this hunk is selected (if any)
		localSelectedRow := -1
		if w.selectedRow >= currentRow && w.selectedRow < currentRow+hunkHeight {
			localSelectedRow = w.selectedRow - currentRow
		}
		hw.SetSelectedRow(localSelectedRow)

		currentRow += hunkHeight + 1 // +1 for spacing
	}

	// Delegate rendering to scroll view
	w.scrollView.Render(buf)
}

// ensureSelectedVisible adjusts scroll offset to keep the selected row visible.
func (w *DiffContentView) ensureSelectedVisible() {
	headerHeight := 2 // File header height
	scrollableHeight := w.Bounds().Height - headerHeight
	if scrollableHeight <= 0 {
		return
	}

	// Convert selected row to content-relative position (subtract header)
	contentRow := w.selectedRow - headerHeight
	if contentRow < 0 {
		contentRow = 0
	}

	offset := w.scrollView.ScrollOffset()

	// Scroll up if selection is above viewport
	if contentRow < offset {
		w.scrollView.SetScrollOffset(contentRow)
	}

	// Scroll down if selection is below viewport
	if contentRow >= offset+scrollableHeight {
		w.scrollView.SetScrollOffset(contentRow - scrollableHeight + 1)
	}
}

// IsRowNavigable returns true if the given row is navigable (a diff line or comment).
func (w *DiffContentView) IsRowNavigable(row int) bool {
	if row < 2 { // Header
		return false
	}

	currentRow := 2
	for hunkIdx, hw := range w.hunks {
		// Skip hunk header
		currentRow++

		for _, line := range hw.hunk.Lines {
			if currentRow == row {
				return true // Diff line
			}
			currentRow++

			if line.NewNum > 0 {
				if conv, exists := w.conversationMap[line.NewNum]; exists {
					commentHeight := calculateCommentHeight(conv)
					if row >= currentRow && row < currentRow+commentHeight {
						return true // Comment row
					}
					currentRow += commentHeight
				}
			}
		}

		if hunkIdx < len(w.hunks)-1 {
			currentRow++ // Spacing
		}
	}
	return false
}

// GetNextNavigableRow returns the next navigable row after the given row.
func (w *DiffContentView) GetNextNavigableRow(currentRow int) int {
	totalHeight := w.CalculateTotalHeight()
	for row := currentRow + 1; row < totalHeight; row++ {
		if w.IsRowNavigable(row) {
			return row
		}
	}
	return currentRow // No next row
}

// GetPrevNavigableRow returns the previous navigable row before the given row.
func (w *DiffContentView) GetPrevNavigableRow(currentRow int) int {
	for row := currentRow - 1; row >= 0; row-- {
		if w.IsRowNavigable(row) {
			return row
		}
	}
	return currentRow // No prev row
}

// GetSourceLineForRow returns the source line number for a given row.
func (w *DiffContentView) GetSourceLineForRow(row int) int {
	if row < 2 {
		return 0
	}

	currentRow := 2
	for hunkIdx, hw := range w.hunks {
		currentRow++ // Hunk header

		for _, line := range hw.hunk.Lines {
			if currentRow == row {
				if line.NewNum > 0 {
					return line.NewNum
				}
				return line.OldNum
			}
			currentRow++

			if line.NewNum > 0 {
				if conv, exists := w.conversationMap[line.NewNum]; exists {
					commentHeight := calculateCommentHeight(conv)
					if row >= currentRow && row < currentRow+commentHeight {
						return line.NewNum // Comment is attached to this line
					}
					currentRow += commentHeight
				}
			}
		}

		if hunkIdx < len(w.hunks)-1 {
			currentRow++ // Spacing
		}
	}
	return 0
}

// GetConversationUUIDForRow returns the conversation UUID for a row (if it's a comment row).
func (w *DiffContentView) GetConversationUUIDForRow(row int) string {
	if row < 2 {
		return ""
	}

	currentRow := 2
	for hunkIdx, hw := range w.hunks {
		currentRow++ // Hunk header

		for _, line := range hw.hunk.Lines {
			currentRow++ // Line

			if line.NewNum > 0 {
				if conv, exists := w.conversationMap[line.NewNum]; exists {
					commentHeight := calculateCommentHeight(conv)
					if row >= currentRow && row < currentRow+commentHeight {
						return conv.UUID
					}
					currentRow += commentHeight
				}
			}
		}

		if hunkIdx < len(w.hunks)-1 {
			currentRow++ // Spacing
		}
	}
	return ""
}

// IsCommentRow returns true if the given row is part of a comment.
func (w *DiffContentView) IsCommentRow(row int) bool {
	return w.GetConversationUUIDForRow(row) != ""
}
