package tui

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/pkg/critic"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/teapot"
	"github.com/charmbracelet/lipgloss"
)

// HunkWidget renders a complete hunk: header + lines + inline comments.
// This is the primary building block of the diff view.
type HunkWidget struct {
	teapot.BaseWidget
	hunk            *ctypes.Hunk
	conversationMap map[int]*critic.Conversation
	highlightedOld  map[int]string // Pre-highlighted content for deleted lines
	highlightedNew  map[int]string // Pre-highlighted content for added lines
	highlightedCtx  map[int]string // Pre-highlighted content for context lines
	selectedRow     int            // Which row within this hunk is selected (-1 = none)
	startRow        int            // Global row number where this hunk starts (for selection mapping)
}

// NewHunkWidget creates a new hunk widget.
func NewHunkWidget(
	hunk *ctypes.Hunk,
	conversationMap map[int]*critic.Conversation,
	highlightedOld, highlightedNew, highlightedCtx map[int]string,
) *HunkWidget {
	w := &HunkWidget{
		BaseWidget:      teapot.NewBaseWidget(teapot.ZOrderDefault),
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
func (w *HunkWidget) calculateHeight() int {
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
func (w *HunkWidget) SetSelectedRow(row int) {
	w.selectedRow = row
}

// SetStartRow sets the global row number where this hunk starts.
func (w *HunkWidget) SetStartRow(row int) {
	w.startRow = row
}

// Hunk returns the underlying hunk.
func (w *HunkWidget) Hunk() *ctypes.Hunk {
	return w.hunk
}

// Render renders the hunk to the buffer.
func (w *HunkWidget) Render(buf *teapot.SubBuffer) {
	width := buf.Width()
	if width <= 0 {
		return
	}

	y := 0

	// Render hunk header
	header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", w.hunk.OldStart, w.hunk.OldLines, w.hunk.NewStart, w.hunk.NewLines)
	if w.hunk.Header != "" {
		header += " " + w.hunk.Header
	}
	w.renderHeaderLine(buf, y, header, width)
	y++

	// Render each line and its comment (if any)
	// Note: Selection highlighting is done via buffer overlay, not here
	for _, line := range w.hunk.Lines {
		// Get highlighted content
		highlighted := w.getHighlightedContent(line)

		// Render the diff line
		w.renderDiffLine(buf, y, line, highlighted, width)
		y++

		// Render comment if exists
		if line.NewNum > 0 {
			if conv, exists := w.conversationMap[line.NewNum]; exists {
				commentHeight := calculateCommentHeight(conv)
				// Check if any row in comment is selected (for hotkey display)
				commentSelected := w.selectedRow >= y && w.selectedRow < y+commentHeight
				w.renderComment(buf, y, conv, width, commentHeight, commentSelected)
				y += commentHeight
			}
		}
	}
}

// getHighlightedContent returns the highlighted content for a line.
func (w *HunkWidget) getHighlightedContent(line *ctypes.Line) string {
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
func (w *HunkWidget) renderHeaderLine(buf *teapot.SubBuffer, y int, header string, width int) {
	if y >= buf.Height() {
		return
	}
	runes := []rune(header)
	for x := 0; x < width; x++ {
		var cell teapot.Cell
		if x < len(runes) {
			cell = teapot.Cell{Rune: runes[x], Style: hunkHeaderStyle}
		} else {
			cell = teapot.Cell{Rune: ' ', Style: hunkHeaderStyle}
		}
		buf.SetCell(x, y, cell)
	}
}

// renderDiffLine renders a single diff line (selection highlighting is done via overlay).
func (w *HunkWidget) renderDiffLine(buf *teapot.SubBuffer, y int, line *ctypes.Line, highlighted string, width int) {
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
	cells := teapot.ParseANSILine(content)
	for x := 0; x < width; x++ {
		var cell teapot.Cell
		if x < len(cells) {
			cell = cells[x]
			cell.Style = cell.Style.Background(bgColor)
		} else {
			cell = teapot.Cell{Rune: ' ', Style: style}
		}
		buf.SetCell(x, y, cell)
	}
}

// renderComment renders an inline comment/conversation.
func (w *HunkWidget) renderComment(buf *teapot.SubBuffer, startY int, conv *critic.Conversation, width, height int, selected bool) {
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
		for x := 0; x < len(animCells) && x < width; x++ {
			buf.SetCell(x, y, animCells[x])
		}
		// Render the rest of the separator (space + dashes)
		if width > 12 {
			buf.SetCell(12, y, teapot.Cell{Rune: ' ', Style: separatorStyle})
		}
		for x := 13; x < width; x++ {
			buf.SetCell(x, y, teapot.Cell{Rune: '-', Style: separatorStyle})
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
		cells := teapot.ParseANSILine(content)
		for x := 0; x < width; x++ {
			var cell teapot.Cell
			if x < len(cells) {
				cell = cells[x]
				cell.Style = cell.Style.Background(lightBlueBg).Foreground(blackFg)
			} else {
				cell = teapot.Cell{Rune: ' ', Style: contentStyle}
			}
			buf.SetCell(x, y, cell)
		}
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

		runes := []rune(bottomLine)
		for x := 0; x < width; x++ {
			var r rune = '-'
			if x < len(runes) {
				r = runes[x]
			}
			buf.SetCell(x, y, teapot.Cell{Rune: r, Style: separatorStyle})
		}
	}
}

// FileHeaderWidget displays the file header.
type FileHeaderWidget struct {
	teapot.BaseWidget
	header string
}

// NewFileHeaderWidget creates a new file header widget.
func NewFileHeaderWidget(file *ctypes.FileDiff) *FileHeaderWidget {
	var header string
	if file.IsDeleted {
		header = file.OldPath + " (deleted)"
	} else if file.IsNew {
		header = file.NewPath + " (new)"
	} else if file.IsRenamed {
		header = file.OldPath + " -> " + file.NewPath + " (renamed)"
	} else {
		header = file.NewPath
	}

	w := &FileHeaderWidget{
		BaseWidget: teapot.NewBaseWidget(teapot.ZOrderDefault),
		header:     header,
	}
	w.SetFocusable(false)
	w.SetConstraints(teapot.DefaultConstraints().WithMinSize(1, 2).WithPreferredSize(0, 2))
	return w
}

// Render renders the file header.
func (w *FileHeaderWidget) Render(buf *teapot.SubBuffer) {
	width := buf.Width()

	runes := []rune(w.header)
	for x := 0; x < width; x++ {
		var cell teapot.Cell
		if x < len(runes) {
			cell = teapot.Cell{Rune: runes[x], Style: hunkHeaderStyle}
		} else {
			cell = teapot.Cell{Rune: ' ', Style: hunkHeaderStyle}
		}
		buf.SetCell(x, 0, cell)
	}

	// Blank second line
	if buf.Height() > 1 {
		emptyStyle := lipgloss.NewStyle()
		for x := 0; x < width; x++ {
			buf.SetCell(x, 1, teapot.Cell{Rune: ' ', Style: emptyStyle})
		}
	}
}

// DiffViewWidget is the main widget that displays the entire diff view.
// It stacks HunkWidgets vertically.
type DiffViewWidget struct {
	teapot.BaseWidget
	file            *ctypes.FileDiff
	hunks           []*HunkWidget
	conversationMap map[int]*critic.Conversation
	highlightedOld  map[int]string
	highlightedNew  map[int]string
	highlightedCtx  map[int]string
	filterMode      FilterMode
	selectedRow     int // Global row number of selected item
	yOffset         int // Scroll offset
}

// NewDiffViewWidget creates a new diff view widget.
func NewDiffViewWidget() *DiffViewWidget {
	w := &DiffViewWidget{
		BaseWidget: teapot.NewBaseWidget(teapot.ZOrderDefault),
	}
	w.SetFocusable(true)
	return w
}

// SetFile sets the file to display and rebuilds the hunk widgets.
func (w *DiffViewWidget) SetFile(
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
	w.yOffset = 0
	w.selectedRow = 3 // First line after header (row 0-1) and first hunk header (row 2)
}

// SetFilterMode sets the filter mode.
func (w *DiffViewWidget) SetFilterMode(mode FilterMode) {
	w.filterMode = mode
}

// rebuildHunks creates HunkWidgets for each hunk.
func (w *DiffViewWidget) rebuildHunks() {
	w.hunks = nil
	if w.file == nil {
		return
	}

	hunksToRender := w.filterHunks(w.file.Hunks)
	currentRow := 2 // After file header

	for _, hunk := range hunksToRender {
		hw := NewHunkWidget(hunk, w.conversationMap, w.highlightedOld, w.highlightedNew, w.highlightedCtx)
		hw.SetStartRow(currentRow)
		w.hunks = append(w.hunks, hw)
		currentRow += hw.calculateHeight() + 1 // +1 for spacing
	}
}

// filterHunks filters hunks based on the current filter mode.
func (w *DiffViewWidget) filterHunks(hunks []*ctypes.Hunk) []*ctypes.Hunk {
	if w.filterMode == FilterModeNone {
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
func (w *DiffViewWidget) hunkMatchesFilter(hunk *ctypes.Hunk) bool {
	for _, line := range hunk.Lines {
		if line.NewNum > 0 {
			if conv, exists := w.conversationMap[line.NewNum]; exists {
				switch w.filterMode {
				case FilterModeWithComments:
					return true
				case FilterModeWithUnresolved:
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
func (w *DiffViewWidget) GetSelectedRow() int {
	return w.selectedRow
}

// SetSelectedRow sets the selected row.
func (w *DiffViewWidget) SetSelectedRow(row int) {
	w.selectedRow = row
}

// GetFile returns the current file.
func (w *DiffViewWidget) GetFile() *ctypes.FileDiff {
	return w.file
}

// CalculateTotalHeight returns the total content height.
func (w *DiffViewWidget) CalculateTotalHeight() int {
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
func (w *DiffViewWidget) Render(buf *teapot.SubBuffer) {
	if w.file == nil {
		return
	}

	width := buf.Width()
	height := buf.Height()

	// Calculate total content height
	contentHeight := w.CalculateTotalHeight()

	// Adjust scroll offset to keep selected item visible
	w.ensureSelectedVisible(height, contentHeight)

	// Render file header (fixed at top)
	fileHeader := NewFileHeaderWidget(w.file)
	headerBuf := buf.Sub(teapot.Rect{X: 0, Y: 0, Width: width, Height: 2})
	fileHeader.Render(headerBuf)

	// Render hunks with scroll offset
	renderY := 2 - w.yOffset

	for hunkIdx, hw := range w.hunks {
		hunkHeight := hw.calculateHeight()

		// Calculate which row within this hunk is selected (if any)
		localSelectedRow := -1
		if w.selectedRow >= renderY+w.yOffset && w.selectedRow < renderY+w.yOffset+hunkHeight {
			localSelectedRow = w.selectedRow - (renderY + w.yOffset)
		}
		hw.SetSelectedRow(localSelectedRow)

		// Render if any part is visible
		if renderY+hunkHeight > 0 && renderY < height {
			hunkBuf := buf.Sub(teapot.Rect{X: 0, Y: renderY, Width: width, Height: hunkHeight})
			hw.Render(hunkBuf)
		}
		renderY += hunkHeight

		// Spacing between hunks
		if hunkIdx < len(w.hunks)-1 {
			if renderY >= 0 && renderY < height {
				emptyStyle := lipgloss.NewStyle()
				for x := 0; x < width; x++ {
					buf.SetCell(x, renderY, teapot.Cell{Rune: ' ', Style: emptyStyle})
				}
			}
			renderY++
		}
	}
}

// ensureSelectedVisible adjusts yOffset to keep the selected row visible.
func (w *DiffViewWidget) ensureSelectedVisible(viewHeight, contentHeight int) {
	// Ensure selected row is in view
	if w.selectedRow < w.yOffset+2 { // +2 for header
		w.yOffset = w.selectedRow - 2
	} else if w.selectedRow >= w.yOffset+viewHeight {
		w.yOffset = w.selectedRow - viewHeight + 1
	}

	// Clamp offset
	maxOffset := contentHeight - viewHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if w.yOffset > maxOffset {
		w.yOffset = maxOffset
	}
	if w.yOffset < 0 {
		w.yOffset = 0
	}
}

// IsRowNavigable returns true if the given row is navigable (a diff line or comment).
func (w *DiffViewWidget) IsRowNavigable(row int) bool {
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
func (w *DiffViewWidget) GetNextNavigableRow(currentRow int) int {
	totalHeight := w.CalculateTotalHeight()
	for row := currentRow + 1; row < totalHeight; row++ {
		if w.IsRowNavigable(row) {
			return row
		}
	}
	return currentRow // No next row
}

// GetPrevNavigableRow returns the previous navigable row before the given row.
func (w *DiffViewWidget) GetPrevNavigableRow(currentRow int) int {
	for row := currentRow - 1; row >= 0; row-- {
		if w.IsRowNavigable(row) {
			return row
		}
	}
	return currentRow // No prev row
}

// GetSourceLineForRow returns the source line number for a given row.
func (w *DiffViewWidget) GetSourceLineForRow(row int) int {
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
func (w *DiffViewWidget) GetConversationUUIDForRow(row int) string {
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
func (w *DiffViewWidget) IsCommentRow(row int) bool {
	return w.GetConversationUUIDForRow(row) != ""
}

