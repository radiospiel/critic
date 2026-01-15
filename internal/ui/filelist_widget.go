package ui

import (
	"fmt"

	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/pkg/critic"
	ctypes "git.15b.it/eno/critic/pkg/types"
	pot "git.15b.it/eno/critic/teapot"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileItem wraps a FileDiff to implement pot.ListItem
type FileItem struct {
	File *ctypes.FileDiff
}

// FilterValue returns the file path for filtering
func (f FileItem) FilterValue() string {
	if f.File.IsDeleted {
		return f.File.OldPath
	}
	return f.File.NewPath
}

// fileAnimInfo tracks animation information for a file indicator
type fileAnimInfo struct {
	bounds pot.Rect
	state  AnimationState
}

// FileListWidget is a teapot-based file list widget.
// FileListWidget implements AnimationWidget for file indicator animations.
type FileListWidget struct {
	pot.BaseWidget
	list            *pot.SelectableList[FileItem]
	messaging       critic.Messaging
	animationTicker *AnimationTicker
	width           int
	height          int
	filterMode      int // 0 = all, 1 = with comments, 2 = unresolved only
	totalFiles      int // Total files before filtering (for "No files match filter" message)

	// Animation support - tracks positions of animated file indicators
	animInfos []fileAnimInfo // Animation info for each visible animated item
}

// NewFileListWidget creates a new file list widget
func NewFileListWidget() *FileListWidget {
	w := &FileListWidget{}

	// Create the SelectableList with a custom renderer
	w.list = pot.NewSelectableList[FileItem](w.renderItem)

	// Configure styles
	w.list.SetStyles(
		selectedFileActiveStyle,
		selectedFileInactiveStyle,
		normalFileStyle,
	)

	return w
}

// renderItem renders a single file item
func (w *FileListWidget) renderItem(buf *pot.SubBuffer, item FileItem, selected bool, focused bool, width int) {
	file := item.File

	// Get the git-relative path for checking conversations
	gitPath := file.NewPath
	if file.IsDeleted {
		gitPath = file.OldPath
	}

	// Get conversation summary from messaging interface
	var hasUnreadAI bool
	var hasUnresolved bool
	var hasResolved bool
	var fileAnimSummary FileAnimationSummary

	if w.messaging != nil {
		summary, err := w.messaging.GetFileConversationSummary(gitPath)
		if err == nil && summary != nil {
			hasUnreadAI = summary.HasUnreadAIMessages
			hasUnresolved = summary.HasUnresolvedComments
			hasResolved = summary.HasResolvedComments
		}

		// Get animation state for this file's conversations
		if hasUnresolved {
			fileAnimSummary = w.getFileAnimationSummary(gitPath)
		}
	}

	// Determine file status symbol
	var status string
	if file.IsNew {
		status = "+"
	} else if file.IsDeleted {
		status = "-"
	} else if file.IsRenamed {
		status = "→"
	} else {
		status = "M"
	}

	// Get file path to display (convert git-relative to cwd-relative)
	path := git.GitPathToDisplayPath(file.NewPath)
	if file.IsDeleted {
		path = git.GitPathToDisplayPath(file.OldPath)
	}

	// Determine left indicator - animation takes priority
	var indicatorStyle lipgloss.Style
	var indicatorRune rune = ' '
	animState := GetFileAnimationState(fileAnimSummary)

	if animState != NoAnimation && w.animationTicker != nil {
		// Animation is active - render placeholder space, actual animation via overlay
		indicatorRune = ' '
		indicatorStyle = lipgloss.NewStyle()

		// Track animation position for RenderInOverlay
		absX, absY := buf.AbsoluteOffset()
		w.animInfos = append(w.animInfos, fileAnimInfo{
			bounds: pot.Rect{X: absX, Y: absY, Width: 1, Height: 1},
			state:  animState,
		})
	} else if hasUnreadAI {
		indicatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
		indicatorRune = '▌'
	} else if hasUnresolved {
		indicatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // Yellow
		indicatorRune = '▌'
	} else if hasResolved {
		indicatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("34")) // Green
		indicatorRune = '▌'
	}

	// Select style based on selection state
	var style lipgloss.Style
	if selected {
		if focused {
			style = selectedFileActiveStyle
		} else {
			style = selectedFileInactiveStyle
		}
	} else {
		style = normalFileStyle
	}

	// Render indicator at column 0
	buf.SetCell(0, 0, pot.Cell{Rune: indicatorRune, Style: indicatorStyle})
	// Space after indicator at column 1
	buf.SetCell(1, 0, pot.Cell{Rune: ' ', Style: style})

	// Render content starting at column 2
	content := fmt.Sprintf("%s %s", status, path)
	availableWidth := width - 2 // -2 for indicator + space

	// Truncate if needed
	runes := []rune(content)
	if len(runes) > availableWidth {
		if availableWidth > 1 {
			runes = append(runes[:availableWidth-1], '…')
		} else if availableWidth > 0 {
			runes = runes[:availableWidth]
		}
	}

	// Write content starting at column 2
	for i, r := range runes {
		buf.SetCell(2+i, 0, pot.Cell{Rune: r, Style: style})
	}

	// Fill remaining width with style (for selection highlight)
	if selected {
		for x := 2 + len(runes); x < width; x++ {
			buf.SetCell(x, 0, pot.Cell{Rune: ' ', Style: style})
		}
	}
}

// getFileAnimationSummary calculates the animation summary for a file
func (w *FileListWidget) getFileAnimationSummary(gitPath string) FileAnimationSummary {
	summary := FileAnimationSummary{}
	if w.messaging == nil {
		return summary
	}

	convs, err := w.messaging.GetConversationsForFile(gitPath)
	if err != nil {
		return summary
	}

	for _, conv := range convs {
		state := GetConversationAnimationState(conv)
		switch state {
		case ThinkingAnimation:
			summary.HasThinking = true
		case LookHereAnimation:
			summary.HasLookHere = true
		}

		// Early exit if both are true
		if summary.HasThinking && summary.HasLookHere {
			return summary
		}
	}

	return summary
}

// SetFiles updates the file list
func (w *FileListWidget) SetFiles(files []*ctypes.FileDiff) {
	items := make([]FileItem, len(files))
	for i, f := range files {
		items[i] = FileItem{File: f}
	}
	w.list.SetItems(items)
}

// GetActiveFile returns the currently selected file
func (w *FileListWidget) GetActiveFile() *ctypes.FileDiff {
	if item, ok := w.list.Selected(); ok {
		return item.File
	}
	return nil
}

// SelectByPath selects a file by its path
func (w *FileListWidget) SelectByPath(path string) bool {
	return w.list.SelectByPredicate(func(item FileItem) bool {
		filePath := item.File.NewPath
		if filePath == "" {
			filePath = item.File.OldPath
		}
		return filePath == path
	})
}

// SelectNext moves to the next file
func (w *FileListWidget) SelectNext() bool {
	idx := w.list.SelectedIndex()
	if idx < len(w.list.Items())-1 {
		w.list.SetSelectedIndex(idx + 1)
		return true
	}
	return false
}

// SelectPrev moves to the previous file
func (w *FileListWidget) SelectPrev() bool {
	idx := w.list.SelectedIndex()
	if idx > 0 {
		w.list.SetSelectedIndex(idx - 1)
		return true
	}
	return false
}

// OnSelect sets a callback for when selection changes
func (w *FileListWidget) OnSelect(fn func(*ctypes.FileDiff)) {
	w.list.OnSelect(func(item FileItem) {
		fn(item.File)
	})
}

// SetMessaging sets the messaging interface
func (w *FileListWidget) SetMessaging(messaging critic.Messaging) {
	w.messaging = messaging
}

// SetAnimationTicker sets the animation ticker for conversation state animations
func (w *FileListWidget) SetAnimationTicker(ticker *AnimationTicker) {
	w.animationTicker = ticker
}

// SetFilterMode sets the current filter mode and total files count
// filterMode: 0 = all, 1 = with comments, 2 = unresolved only
func (w *FileListWidget) SetFilterMode(filterMode int, totalFiles int) {
	w.filterMode = filterMode
	w.totalFiles = totalFiles
}

// SetBounds implements pot.Widget
func (w *FileListWidget) SetBounds(bounds pot.Rect) {
	w.BaseWidget.SetBounds(bounds)
	w.width = bounds.Width
	w.height = bounds.Height
	w.list.SetBounds(bounds)
}

// SetFocused implements pot.Widget
func (w *FileListWidget) SetFocused(focused bool) {
	w.BaseWidget.SetFocused(focused)
	w.list.SetFocused(focused)
}

// Render implements pot.Widget.
// Animation indicators are rendered as placeholders (spaces); actual animation
// is rendered via RenderInOverlay.
func (w *FileListWidget) Render(buf *pot.SubBuffer) {
	// Reset animation tracking for this render cycle
	w.animInfos = nil

	if len(w.list.Items()) == 0 {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#666"})
		// Show appropriate message based on filter mode
		var msg string
		if w.filterMode > 0 && w.totalFiles > 0 {
			// Files exist but none match the current filter
			msg = "No files match filter (press f to show all)"
		} else {
			msg = "No files changed"
		}
		buf.SetString(1, 1, msg, style)
		return
	}
	w.list.Render(buf)
}

// HandleKey implements pot.Widget
func (w *FileListWidget) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	return w.list.HandleKey(msg)
}

// Children returns the child widgets
func (w *FileListWidget) Children() []pot.Widget {
	return nil
}

// View returns a string representation (for compatibility with existing code)
func (w *FileListWidget) View() string {
	buf := pot.NewBuffer(w.width, w.height)
	sub := buf.Sub(buf.Bounds())
	w.Render(sub)

	// Apply animation overlay
	if w.NeedsAnimation() {
		w.RenderInOverlay(buf)
	}

	return buf.String()
}

// SetSize sets the size (for compatibility)
func (w *FileListWidget) SetSize(width, height int) {
	w.SetBounds(pot.NewRect(0, 0, width, height))
}

// AnimationBounds returns the screen-space bounds for all animation regions.
// Implements teapot.AnimationWidget.
func (w *FileListWidget) AnimationBounds() []pot.Rect {
	bounds := make([]pot.Rect, len(w.animInfos))
	for i, info := range w.animInfos {
		bounds[i] = info.bounds
	}
	return bounds
}

// NeedsAnimation returns true if this widget has active animations.
// Implements teapot.AnimationWidget.
func (w *FileListWidget) NeedsAnimation() bool {
	return w.animationTicker != nil && len(w.animInfos) > 0
}

// RenderInOverlay renders the animation indicators directly to the buffer.
// Implements teapot.AnimationWidget.
func (w *FileListWidget) RenderInOverlay(buf *pot.Buffer) {
	if w.animationTicker == nil {
		return
	}

	// Render each animation at its tracked position
	for _, info := range w.animInfos {
		// Check bounds are within buffer
		if info.bounds.Y < 0 || info.bounds.Y >= buf.Height() {
			continue
		}
		if info.bounds.X < 0 || info.bounds.X >= buf.Width() {
			continue
		}

		// Get the animation character and style for this state
		indicatorRune := w.animationTicker.GetFrameRune(info.state)
		indicatorStyle := w.animationTicker.GetFrameStyle(info.state)

		// Render the animation character
		buf.SetCell(info.bounds.X, info.bounds.Y, pot.Cell{
			Rune:  indicatorRune,
			Style: indicatorStyle,
		})
	}
}
