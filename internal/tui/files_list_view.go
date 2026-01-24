package tui

import (
	"fmt"
	"slices"
	"strings"

	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/session"
	"git.15b.it/eno/critic/pkg/critic"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/simple-go/observable"
	"git.15b.it/eno/critic/simple-go/preconditions"
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

// FilesListView is a teapot-based file listView widget.
type FilesListView struct {
	pot.BaseView
	listView      *pot.ListView[FileItem]
	messaging     critic.Messaging
	session       *session.Session
	subscriptions []observable.Subscription
	width         int
	height        int
	filterMode    int // 0 = all, 1 = with comments, 2 = unresolved only
	totalFiles    int // Total files before filtering (for "No files match filter" message)
}

// NewFilesListView creates a new file listView widget
func NewFilesListView(ses *session.Session, messaging critic.Messaging) *FilesListView {
	preconditions.Check(ses != nil, "ses must be set")
	preconditions.Check(messaging != nil, "messaging must be set")
	w := &FilesListView{session: ses, messaging: messaging}

	// Create the List with a custom renderer
	w.listView = pot.NewSelectableList[FileItem](w.renderItem)

	// Configure styles
	w.listView.SetStyles(
		selectedFileActiveStyle,
		selectedFileInactiveStyle,
		normalFileStyle,
	)

	// Register selection change callback to update ses
	w.listView.OnSelectionChange(
		func(index int, item *FileItem) {
			preconditions.Check(item != nil, "item must be set")

			filePath := item.File.NewPath
			if filePath == "" {
				filePath = item.File.OldPath
			}

			ses.Transaction(func(txn *observable.Txn) {
				txn.SetValueAtKey(session.KeySelectedFileIndex, index)
				txn.SetValueAtKey(session.KeySelectedFilePath, filePath)
			})
		})

	return w
}

// renderItem renders a single file item
func (w *FilesListView) renderItem(buf *pot.SubBuffer, item FileItem, selected bool, focused bool, width int) {
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

	if fileAnimSummary.HasAnimation() {
		indicatorRune, indicatorStyle = GetAnimatedIndicator(fileAnimSummary.HasThinking, fileAnimSummary.HasLookHere)
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

	// Render indicator at column 0 with selection background if selected
	if selected {
		// Combine indicator foreground with selection background
		combinedStyle := style.Foreground(indicatorStyle.GetForeground())
		buf.SetString(0, 0, string(indicatorRune), combinedStyle)
	} else {
		buf.SetString(0, 0, string(indicatorRune), indicatorStyle)
	}

	// Render content
	contentStart := 1
	availableWidth := width - 1 // -1 for indicator
	content := fmt.Sprintf("%s %s", status, path)

	// Truncate if needed
	runes := []rune(content)
	if len(runes) > availableWidth {
		if availableWidth > 1 {
			runes = append(runes[:availableWidth-1], '…')
		} else if availableWidth > 0 {
			runes = runes[:availableWidth]
		}
	}

	// Write content
	buf.SetString(contentStart, 0, string(runes), style)

	// Fill remaining width with style (for selection highlight)
	if selected {
		contentEnd := contentStart + len(runes)
		remaining := width - contentEnd
		if remaining > 0 {
			buf.SetString(contentEnd, 0, strings.Repeat(" ", remaining), style)
		}
	}
}

// getFileAnimationSummary calculates the animation summary for a file
func (w *FilesListView) getFileAnimationSummary(gitPath string) FileAnimationSummary {
	var summary FileAnimationSummary
	if w.messaging == nil {
		return summary
	}

	convs, err := w.messaging.GetConversationsForFile(gitPath)
	if err != nil {
		return summary
	}

	for _, conv := range convs {
		summary.UpdateFromConversation(conv)
		// Early exit if both are true
		if summary.HasThinking && summary.HasLookHere {
			return summary
		}
	}

	return summary
}

// SetFiles updates the file listView
func (w *FilesListView) SetFiles(files []*ctypes.FileDiff) {
	items := make([]FileItem, len(files))
	for i, f := range files {
		items[i] = FileItem{File: f}
	}
	w.listView.SetItems(items)
	w.Repaint() // Propagate dirty state to parent (MainView)
}

// GetActiveFile returns the currently selected file
func (w *FilesListView) GetActiveFile() *ctypes.FileDiff {
	if item, ok := w.listView.Selected(); ok {
		return item.File
	}
	return nil
}

// SelectByPath selects a file by its path
func (w *FilesListView) SelectByPath(path string) bool {
	index := slices.IndexFunc(w.listView.Items(), func(item FileItem) bool {
		filePath := item.File.NewPath
		if filePath == "" {
			filePath = item.File.OldPath
		}
		return filePath == path
	})

	return w.SetSelectedIndex(index)
}

func (w *FilesListView) SetSelectedIndex(idx int) bool {
	if idx < 0 || idx >= len(w.listView.Items())-1 {
		return false
	}

	// ListView.SetSelectedIndex will trigger onSelectionChange callback
	w.listView.SetSelectedIndex(idx)
	return true
}

// SelectNext moves to the next file
func (w *FilesListView) SelectNext() bool {
	idx := w.listView.SelectedIndex()
	return w.SetSelectedIndex(idx + 1)
}

// SelectPrev moves to the previous file
func (w *FilesListView) SelectPrev() bool {
	idx := w.listView.SelectedIndex()
	return w.SetSelectedIndex(idx - 1)
}

// SetSession sets the session and subscribes to relevant keys.
// The view will automatically update when these session keys change:
// - diff.files: Updates the file listView
// - tui.fileIndex: Updates selection
// - tui.filePath: Updates selection by path
// - tui.focusedPane: Updates focus state
func (w *FilesListView) SetSession(s *session.Session) {
	// Clear previous subscriptions
	if w.session != nil && len(w.subscriptions) > 0 {
		w.session.ClearSubscriptions(w.subscriptions...)
		w.subscriptions = nil
	}

	w.session = s
	if s == nil {
		return
	}

	// Subscribe to diff.files changes
	filesSub := s.OnKeyChange(session.KeyFiles, func(key string) {
		files := s.GetFiles()
		w.updateFilesFromSession(files)
	})
	w.subscriptions = append(w.subscriptions, filesSub)

	// Subscribe to tui.fileIndex changes
	indexSub := s.OnKeyChange(session.KeySelectedFileIndex, func(key string) {
		index := s.GetSelectedFileIndex()
		w.updateSelectionFromSession(index)
	})
	w.subscriptions = append(w.subscriptions, indexSub)

	// Subscribe to tui.focusedPane changes
	focusSub := s.OnKeyChange(session.KeyFocusedPane, func(key string) {
		pane := s.GetFocusedPane()
		focused := pane == "fileList"
		if w.Focused() != focused {
			w.SetFocused(focused)
			w.Repaint()
		}
	})
	w.subscriptions = append(w.subscriptions, focusSub)
}

// updateFilesFromSession updates the file listView from session data
func (w *FilesListView) updateFilesFromSession(files []*ctypes.FileDiff) {
	items := make([]FileItem, len(files))
	for i, f := range files {
		items[i] = FileItem{File: f}
	}
	w.listView.SetItems(items)
	w.totalFiles = len(files)
	w.Repaint()
}

// updateSelectionFromSession updates the selection from session data
func (w *FilesListView) updateSelectionFromSession(index int) {
	if w.listView.SelectedIndex() != index {
		w.listView.SetSelectedIndex(index)
		w.Repaint()
	}
}

// ClearSubscriptions clears all session subscriptions
func (w *FilesListView) ClearSubscriptions() {
	if w.session != nil && len(w.subscriptions) > 0 {
		w.session.ClearSubscriptions(w.subscriptions...)
		w.subscriptions = nil
	}
}

// SetFilterMode sets the current filter mode and total files count
// filterMode: 0 = all, 1 = with comments, 2 = unresolved only
func (w *FilesListView) SetFilterMode(filterMode int, totalFiles int) {
	w.filterMode = filterMode
	w.totalFiles = totalFiles
}

// GetFilterInfo returns the current filtered file count and total file count
// This is used by the status bar to avoid re-filtering on every render
func (w *FilesListView) GetFilterInfo() (filteredCount, totalCount int) {
	return len(w.listView.Items()), w.totalFiles
}

// SetBounds implements pot.View
func (w *FilesListView) SetBounds(bounds pot.Rect) {
	w.BaseView.SetBounds(bounds)
	w.width = bounds.Width
	w.height = bounds.Height
	w.listView.SetBounds(bounds)
}

// SetFocused implements pot.View
func (w *FilesListView) SetFocused(focused bool) {
	w.BaseView.SetFocused(focused)
	w.listView.SetFocused(focused)
}

// Render implements pot.View.
func (w *FilesListView) Render(buf *pot.SubBuffer) {
	if len(w.listView.Items()) == 0 {
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
	w.listView.Render(buf)
}

// HandleKey implements pot.View
func (w *FilesListView) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	return w.listView.HandleKey(msg)
}

// Children returns the child widgets
func (w *FilesListView) Children() []pot.View {
	return nil
}

// AcceptsFocus returns true as the file listView can receive focus.
func (w *FilesListView) AcceptsFocus() bool {
	return true
}

// FocusNext is a no-op for file listView as it has no focusable children.
func (w *FilesListView) FocusNext() bool {
	return false
}

// FocusPrev is a no-op for file listView as it has no focusable children.
func (w *FilesListView) FocusPrev() bool {
	return false
}

// SetSize sets the size (for compatibility)
func (w *FilesListView) SetSize(width, height int) {
	w.SetBounds(pot.NewRect(0, 0, width, height))
}
