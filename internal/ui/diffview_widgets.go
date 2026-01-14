package ui

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/pkg/critic"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/teapot"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiffLineWidget displays a single diff line with line number, indicator, and content.
type DiffLineWidget struct {
	teapot.BaseWidget
	line               *ctypes.Line
	highlightedContent string
	lineType           ctypes.LineType
	selected           bool // Whether this line is selected (cursor on it)
	sourceLine         int  // Source line number for comment association
}

// NewDiffLineWidget creates a new diff line widget.
func NewDiffLineWidget(line *ctypes.Line, highlightedContent string) *DiffLineWidget {
	w := &DiffLineWidget{
		BaseWidget:         teapot.NewBaseWidget(),
		line:               line,
		highlightedContent: highlightedContent,
		lineType:           line.Type,
	}
	w.SetFocusable(true)
	// Set constraints - a single line widget has fixed height of 1
	w.SetConstraints(teapot.DefaultConstraints().WithMinSize(1, 1).WithPreferredSize(0, 1))

	// Track source line number
	if line.NewNum > 0 {
		w.sourceLine = line.NewNum
	} else {
		w.sourceLine = line.OldNum
	}
	return w
}

// SetSelected sets whether this line is selected.
func (w *DiffLineWidget) SetSelected(selected bool) {
	w.selected = selected
}

// Selected returns whether this line is selected.
func (w *DiffLineWidget) Selected() bool {
	return w.selected
}

// SourceLine returns the source line number.
func (w *DiffLineWidget) SourceLine() int {
	return w.sourceLine
}

// Line returns the underlying diff line.
func (w *DiffLineWidget) Line() *ctypes.Line {
	return w.line
}

// Render renders the diff line to the buffer.
func (w *DiffLineWidget) Render(buf *teapot.SubBuffer) {
	width := buf.Width()
	if width <= 0 {
		return
	}

	// Build line number prefix: " 123 + "
	var lineNum int
	var indicator string
	switch w.lineType {
	case ctypes.LineAdded:
		lineNum = w.line.NewNum
		indicator = "+"
	case ctypes.LineDeleted:
		lineNum = w.line.OldNum
		indicator = "-"
	case ctypes.LineContext:
		lineNum = w.line.NewNum
		indicator = " "
	}

	prefix := fmt.Sprintf("%4d %s ", lineNum, indicator)

	// Get background color for this line type
	var bgColor lipgloss.Color
	switch w.lineType {
	case ctypes.LineAdded:
		bgColor = lipgloss.Color("22") // Dark green
	case ctypes.LineDeleted:
		bgColor = lipgloss.Color("52") // Dark red
	default:
		bgColor = lipgloss.Color("0") // Black
	}

	style := lipgloss.NewStyle().Background(bgColor)
	if w.selected {
		style = style.Reverse(true)
	}

	// Render prefix and content
	content := prefix + w.highlightedContent

	// Parse ANSI and render cell by cell
	cells := teapot.ParseANSILine(content)
	for x := 0; x < width; x++ {
		var cell teapot.Cell
		if x < len(cells) {
			cell = cells[x]
			// Apply background and selection
			cell.Style = cell.Style.Background(bgColor)
			if w.selected {
				cell.Style = cell.Style.Reverse(true)
			}
		} else {
			// Padding
			cell = teapot.Cell{Rune: ' ', Style: style}
		}
		buf.SetCell(x, 0, cell)
	}
}

// HandleKey handles key events (no-op for line widgets, handled by parent).
func (w *DiffLineWidget) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	return false, nil
}

// CommentWidget displays an inline conversation preview.
type CommentWidget struct {
	teapot.BaseWidget
	conversation    *critic.Conversation
	selected        bool
	animationTicker *AnimationTicker
	sourceLine      int      // Source line this comment is attached to
	contentLines    []string // Cached content lines
}

// NewCommentWidget creates a new comment widget for a conversation.
func NewCommentWidget(conv *critic.Conversation, ticker *AnimationTicker) *CommentWidget {
	w := &CommentWidget{
		BaseWidget:      teapot.NewBaseWidget(),
		conversation:    conv,
		animationTicker: ticker,
		sourceLine:      conv.LineNumber,
	}
	w.SetFocusable(true)
	// Build and cache content lines
	w.contentLines = w.buildContentLines()
	// Calculate height based on content: top separator + content + bottom separator
	height := 1 + len(w.contentLines) + 1
	w.SetConstraints(teapot.DefaultConstraints().WithMinSize(1, height).WithPreferredSize(0, height))
	return w
}

// buildContentLines builds the content lines for the conversation.
func (w *CommentWidget) buildContentLines() []string {
	var allLines []string

	for i, msg := range w.conversation.Messages {
		prefix := "You"
		if msg.Author == critic.AuthorAI {
			prefix = "AI"
		}

		if i == 0 && msg.Author == critic.AuthorHuman {
			// First human message without prefix
			msgLines := strings.Split(msg.Message, "\n")
			for _, line := range msgLines {
				allLines = append(allLines, renderMarkdown(line))
			}
		} else {
			// Replies with prefix
			replyLines := strings.Split(msg.Message, "\n")
			for j, line := range replyLines {
				if j == 0 {
					allLines = append(allLines, fmt.Sprintf("%s: %s", prefix, renderMarkdown(line)))
				} else {
					indent := strings.Repeat(" ", len(prefix)+2)
					allLines = append(allLines, indent+renderMarkdown(line))
				}
			}
		}
	}

	// Prepend resolved status
	if w.conversation.Status == critic.StatusResolved && len(allLines) > 0 {
		allLines[0] = "(Resolved) " + allLines[0]
	}

	return allLines
}

// SetSelected sets whether this comment widget is selected.
func (w *CommentWidget) SetSelected(selected bool) {
	w.selected = selected
}

// Selected returns whether this comment widget is selected.
func (w *CommentWidget) Selected() bool {
	return w.selected
}

// Conversation returns the underlying conversation.
func (w *CommentWidget) Conversation() *critic.Conversation {
	return w.conversation
}

// SourceLine returns the source line this comment is attached to.
func (w *CommentWidget) SourceLine() int {
	return w.sourceLine
}

// Render renders the comment widget to the buffer.
func (w *CommentWidget) Render(buf *teapot.SubBuffer) {
	width := buf.Width()
	height := buf.Height()
	if width <= 0 || height <= 0 {
		return
	}

	// Styles
	lightBlueBg := lipgloss.Color("#6B95D8")
	blackFg := lipgloss.Color("0")
	grayFg := lipgloss.Color("240")

	contentStyle := lipgloss.NewStyle().Background(lightBlueBg).Foreground(blackFg)
	separatorStyle := lipgloss.NewStyle().Foreground(grayFg)

	y := 0

	// Top separator with snake animation
	if y < height {
		var animFrame string
		if w.animationTicker != nil {
			animFrame = w.animationTicker.GetSeparatorFrame()
		} else {
			animFrame = strings.Repeat(" ", 12)
		}
		dashCount := width - 13
		if dashCount < 0 {
			dashCount = 0
		}
		separatorLine := animFrame + " " + strings.Repeat("-", dashCount)
		runes := []rune(separatorLine)
		for x := 0; x < width; x++ {
			var r rune = ' '
			if x < len(runes) {
				r = runes[x]
			}
			buf.SetCell(x, y, teapot.Cell{Rune: r, Style: separatorStyle})
		}
		y++
	}

	// Content lines
	for _, line := range w.contentLines {
		if y >= height-1 { // Leave room for bottom separator
			break
		}

		lineStyle := contentStyle
		if w.selected {
			lineStyle = lineStyle.Reverse(true)
		}

		content := " " + line
		cells := teapot.ParseANSILine(content)
		for x := 0; x < width; x++ {
			var cell teapot.Cell
			if x < len(cells) {
				cell = cells[x]
				cell.Style = cell.Style.Background(lightBlueBg).Foreground(blackFg)
				if w.selected {
					cell.Style = cell.Style.Reverse(true)
				}
			} else {
				cell = teapot.Cell{Rune: ' ', Style: lineStyle}
			}
			buf.SetCell(x, y, cell)
		}
		y++
	}

	// Bottom separator with hotkeys if selected
	if y < height {
		var separatorText string
		if w.selected {
			separatorText = "[R]esolve - [Enter] reply"
			if w.conversation.Status == critic.StatusResolved {
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

// HandleKey handles key events (no-op for comment widgets, handled by parent).
func (w *CommentWidget) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	return false, nil
}

// HunkHeaderWidget displays the hunk header line (@@ -a,b +c,d @@).
type HunkHeaderWidget struct {
	teapot.BaseWidget
	header string
}

// NewHunkHeaderWidget creates a new hunk header widget.
func NewHunkHeaderWidget(hunk *ctypes.Hunk) *HunkHeaderWidget {
	header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
	if hunk.Header != "" {
		header += " " + hunk.Header
	}

	w := &HunkHeaderWidget{
		BaseWidget: teapot.NewBaseWidget(),
		header:     header,
	}
	w.SetFocusable(false)
	w.SetConstraints(teapot.DefaultConstraints().WithMinSize(1, 1).WithPreferredSize(0, 1))
	return w
}

// Render renders the hunk header.
func (w *HunkHeaderWidget) Render(buf *teapot.SubBuffer) {
	width := buf.Width()
	style := hunkHeaderStyle

	runes := []rune(w.header)
	for x := 0; x < width; x++ {
		var cell teapot.Cell
		if x < len(runes) {
			cell = teapot.Cell{Rune: runes[x], Style: style}
		} else {
			cell = teapot.Cell{Rune: ' ', Style: style}
		}
		buf.SetCell(x, 0, cell)
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
		BaseWidget: teapot.NewBaseWidget(),
		header:     header,
	}
	w.SetFocusable(false)
	// Header + blank line = 2 lines
	w.SetConstraints(teapot.DefaultConstraints().WithMinSize(1, 2).WithPreferredSize(0, 2))
	return w
}

// Render renders the file header.
func (w *FileHeaderWidget) Render(buf *teapot.SubBuffer) {
	width := buf.Width()
	style := hunkHeaderStyle

	// Render first line with header
	runes := []rune(w.header)
	for x := 0; x < width; x++ {
		var cell teapot.Cell
		if x < len(runes) {
			cell = teapot.Cell{Rune: runes[x], Style: style}
		} else {
			cell = teapot.Cell{Rune: ' ', Style: style}
		}
		buf.SetCell(x, 0, cell)
	}

	// Render blank second line
	if buf.Height() > 1 {
		emptyStyle := lipgloss.NewStyle()
		for x := 0; x < width; x++ {
			buf.SetCell(x, 1, teapot.Cell{Rune: ' ', Style: emptyStyle})
		}
	}
}

// SpacerLineWidget is a single blank line for spacing between hunks.
type SpacerLineWidget struct {
	teapot.BaseWidget
}

// NewSpacerLineWidget creates a new spacer line widget.
func NewSpacerLineWidget() *SpacerLineWidget {
	w := &SpacerLineWidget{
		BaseWidget: teapot.NewBaseWidget(),
	}
	w.SetFocusable(false)
	w.SetConstraints(teapot.DefaultConstraints().WithMinSize(1, 1).WithPreferredSize(0, 1))
	return w
}

// Render renders the spacer line (empty).
func (w *SpacerLineWidget) Render(buf *teapot.SubBuffer) {
	width := buf.Width()
	style := lipgloss.NewStyle()
	for x := 0; x < width; x++ {
		buf.SetCell(x, 0, teapot.Cell{Rune: ' ', Style: style})
	}
}

// SelectableItem is an interface for items that can be selected in the diff view.
type SelectableItem interface {
	teapot.Widget
	SetSelected(bool)
	Selected() bool
	SourceLine() int
}

// Ensure widgets implement SelectableItem
var _ SelectableItem = (*DiffLineWidget)(nil)
var _ SelectableItem = (*CommentWidget)(nil)

// DiffViewWidget is the main widget that displays the entire diff view as a vertical layout.
// It contains a file header and stacks of hunk displays.
type DiffViewWidget struct {
	teapot.BaseWidget
	file              *ctypes.FileDiff
	selectableItems   []SelectableItem // All selectable items in order
	selectedIndex     int              // Index of currently selected item
	yOffset           int              // Scroll offset
	conversationMap   map[int]*critic.Conversation
	highlightedOld    map[int]string
	highlightedNew    map[int]string
	highlightedCtx    map[int]string
	animationTicker   *AnimationTicker
	messaging         critic.Messaging
	filterMode        FilterMode
}

// NewDiffViewWidget creates a new diff view widget.
func NewDiffViewWidget() *DiffViewWidget {
	w := &DiffViewWidget{
		BaseWidget:      teapot.NewBaseWidget(),
		selectableItems: make([]SelectableItem, 0),
	}
	w.SetFocusable(true)
	return w
}

// SetFile sets the file to display and rebuilds the widget tree.
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
	w.rebuildSelectableItems()
	w.yOffset = 0
	if len(w.selectableItems) > 0 {
		w.selectedIndex = 0
		w.selectableItems[0].SetSelected(true)
	}
}

// SetAnimationTicker sets the animation ticker.
func (w *DiffViewWidget) SetAnimationTicker(ticker *AnimationTicker) {
	w.animationTicker = ticker
}

// SetMessaging sets the messaging interface.
func (w *DiffViewWidget) SetMessaging(messaging critic.Messaging) {
	w.messaging = messaging
}

// SetFilterMode sets the filter mode.
func (w *DiffViewWidget) SetFilterMode(mode FilterMode) {
	w.filterMode = mode
}

// rebuildSelectableItems rebuilds the list of selectable items.
func (w *DiffViewWidget) rebuildSelectableItems() {
	w.selectableItems = make([]SelectableItem, 0)

	if w.file == nil {
		return
	}

	// Filter hunks based on filter mode
	hunksToRender := w.filterHunks(w.file.Hunks)

	for _, hunk := range hunksToRender {
		for _, line := range hunk.Lines {
			// Get highlighted content
			var highlighted string
			switch line.Type {
			case ctypes.LineAdded:
				if hl, ok := w.highlightedNew[line.NewNum]; ok {
					highlighted = hl
				} else {
					highlighted = line.Content
				}
			case ctypes.LineDeleted:
				if hl, ok := w.highlightedOld[line.OldNum]; ok {
					highlighted = hl
				} else {
					highlighted = line.Content
				}
			case ctypes.LineContext:
				if hl, ok := w.highlightedCtx[line.NewNum]; ok {
					highlighted = hl
				} else {
					highlighted = line.Content
				}
			default:
				highlighted = line.Content
			}

			// Create line widget
			lineWidget := NewDiffLineWidget(line, highlighted)
			w.selectableItems = append(w.selectableItems, lineWidget)

			// Check for conversation at this line
			if line.NewNum > 0 {
				if conv, exists := w.conversationMap[line.NewNum]; exists {
					commentWidget := NewCommentWidget(conv, w.animationTicker)
					w.selectableItems = append(w.selectableItems, commentWidget)
				}
			}
		}
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

// hunkMatchesFilter checks if a hunk should be included based on the current filter mode.
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

// SelectedItem returns the currently selected item.
func (w *DiffViewWidget) SelectedItem() SelectableItem {
	if w.selectedIndex >= 0 && w.selectedIndex < len(w.selectableItems) {
		return w.selectableItems[w.selectedIndex]
	}
	return nil
}

// SelectNext moves selection to the next item.
func (w *DiffViewWidget) SelectNext() bool {
	if w.selectedIndex < len(w.selectableItems)-1 {
		if w.selectedIndex >= 0 {
			w.selectableItems[w.selectedIndex].SetSelected(false)
		}
		w.selectedIndex++
		w.selectableItems[w.selectedIndex].SetSelected(true)
		return true
	}
	return false
}

// SelectPrev moves selection to the previous item.
func (w *DiffViewWidget) SelectPrev() bool {
	if w.selectedIndex > 0 {
		w.selectableItems[w.selectedIndex].SetSelected(false)
		w.selectedIndex--
		w.selectableItems[w.selectedIndex].SetSelected(true)
		return true
	}
	return false
}

// GetSourceLine returns the source line of the currently selected item.
func (w *DiffViewWidget) GetSourceLine() int {
	if item := w.SelectedItem(); item != nil {
		return item.SourceLine()
	}
	return 0
}

// IsAtTop returns true if at the first selectable item.
func (w *DiffViewWidget) IsAtTop() bool {
	return w.selectedIndex <= 0
}

// IsAtBottom returns true if at the last selectable item.
func (w *DiffViewWidget) IsAtBottom() bool {
	return w.selectedIndex >= len(w.selectableItems)-1
}

// GetFile returns the current file.
func (w *DiffViewWidget) GetFile() *ctypes.FileDiff {
	return w.file
}

// Render renders the entire diff view.
func (w *DiffViewWidget) Render(buf *teapot.SubBuffer) {
	if w.file == nil {
		return
	}

	width := buf.Width()
	height := buf.Height()

	// Filter hunks
	hunksToRender := w.filterHunks(w.file.Hunks)

	// Calculate total content height for scroll management
	contentHeight := 2 // File header height
	for hunkIdx, hunk := range hunksToRender {
		contentHeight++ // hunk header
		for _, line := range hunk.Lines {
			contentHeight++ // line
			if line.NewNum > 0 {
				if conv, exists := w.conversationMap[line.NewNum]; exists {
					// Comment height: separator + content + separator
					commentWidget := NewCommentWidget(conv, w.animationTicker)
					contentHeight += commentWidget.Constraints().EffectivePreferredHeight()
				}
			}
		}
		if hunkIdx < len(hunksToRender)-1 {
			contentHeight++ // spacing
		}
	}

	// Adjust scroll offset to keep selected item visible
	w.ensureSelectedVisible(height, contentHeight)

	// Render file header at the top (not scrolled)
	fileHeader := NewFileHeaderWidget(w.file)
	headerBuf := buf.Sub(teapot.Rect{X: 0, Y: 0, Width: width, Height: 2})
	fileHeader.Render(headerBuf)

	// Render hunks with scroll offset
	renderY := 2 - w.yOffset // Start after header, adjusted for scroll
	itemIndex := 0

	for hunkIdx, hunk := range hunksToRender {
		// Render hunk header
		if renderY >= 0 && renderY < height {
			hunkHeader := NewHunkHeaderWidget(hunk)
			hunkBuf := buf.Sub(teapot.Rect{X: 0, Y: renderY, Width: width, Height: 1})
			hunkHeader.Render(hunkBuf)
		}
		renderY++

		// Render lines and comments
		for _, line := range hunk.Lines {
			// Find corresponding selectable item
			var lineWidget *DiffLineWidget
			var commentWidget *CommentWidget

			if itemIndex < len(w.selectableItems) {
				if lw, ok := w.selectableItems[itemIndex].(*DiffLineWidget); ok {
					lineWidget = lw
					itemIndex++
				}
			}

			// Render line
			if renderY >= 0 && renderY < height && lineWidget != nil {
				lineBuf := buf.Sub(teapot.Rect{X: 0, Y: renderY, Width: width, Height: 1})
				lineWidget.Render(lineBuf)
			}
			renderY++

			// Check for comment
			if line.NewNum > 0 {
				if _, exists := w.conversationMap[line.NewNum]; exists {
					if itemIndex < len(w.selectableItems) {
						if cw, ok := w.selectableItems[itemIndex].(*CommentWidget); ok {
							commentWidget = cw
							itemIndex++
						}
					}

					if commentWidget != nil {
						commentHeight := commentWidget.Constraints().EffectivePreferredHeight()
						// Render comment if any part is visible
						if renderY+commentHeight > 0 && renderY < height {
							visibleStart := renderY
							if visibleStart < 0 {
								visibleStart = 0
							}
							visibleHeight := commentHeight
							if renderY+commentHeight > height {
								visibleHeight = height - renderY
							}
							if visibleHeight > 0 {
								commentBuf := buf.Sub(teapot.Rect{X: 0, Y: renderY, Width: width, Height: commentHeight})
								commentWidget.Render(commentBuf)
							}
						}
						renderY += commentHeight
					}
				}
			}
		}

		// Spacing between hunks
		if hunkIdx < len(hunksToRender)-1 {
			if renderY >= 0 && renderY < height {
				spacerBuf := buf.Sub(teapot.Rect{X: 0, Y: renderY, Width: width, Height: 1})
				spacer := NewSpacerLineWidget()
				spacer.Render(spacerBuf)
			}
			renderY++
		}
	}
}

// ensureSelectedVisible adjusts yOffset to keep the selected item visible.
func (w *DiffViewWidget) ensureSelectedVisible(viewHeight, contentHeight int) {
	if len(w.selectableItems) == 0 {
		return
	}

	// Calculate the y position of the selected item
	selectedY := w.calculateItemY(w.selectedIndex)
	selectedHeight := 1
	if item := w.SelectedItem(); item != nil {
		selectedHeight = item.Constraints().EffectivePreferredHeight()
		if selectedHeight == 0 {
			selectedHeight = 1
		}
	}

	// Adjust offset to keep item in view
	if selectedY < w.yOffset {
		w.yOffset = selectedY
	} else if selectedY+selectedHeight > w.yOffset+viewHeight {
		w.yOffset = selectedY + selectedHeight - viewHeight
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

// calculateItemY calculates the y position of an item given its index.
func (w *DiffViewWidget) calculateItemY(index int) int {
	y := 2 // File header height

	hunksToRender := w.filterHunks(w.file.Hunks)
	itemIndex := 0

	for hunkIdx, hunk := range hunksToRender {
		y++ // Hunk header

		for _, line := range hunk.Lines {
			if itemIndex == index {
				return y
			}
			y++ // Line
			itemIndex++

			// Check for comment
			if line.NewNum > 0 {
				if conv, exists := w.conversationMap[line.NewNum]; exists {
					if itemIndex == index {
						return y
					}
					commentWidget := NewCommentWidget(conv, w.animationTicker)
					y += commentWidget.Constraints().EffectivePreferredHeight()
					itemIndex++
				}
			}
		}

		if hunkIdx < len(hunksToRender)-1 {
			y++ // Spacing
		}
	}

	return y
}

// HandleKey handles keyboard input.
func (w *DiffViewWidget) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if w.SelectPrev() {
			return true, nil
		}
	case "down", "j":
		if w.SelectNext() {
			return true, nil
		}
	}
	return false, nil
}
