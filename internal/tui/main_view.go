package tui

import (
	"git.15b.it/eno/critic/simple-go/logger"
	pot "git.15b.it/eno/critic/teapot"
	"github.com/charmbracelet/lipgloss"
)

// MainView is the root layout widget containing the main content and status bar.
// Structure: VBox with [HSplit(fileList, diffView), StatusBar]
type MainView struct {
	pot.BaseView

	// Child widgets
	fileList  *FilesListView
	diffView  *DiffView
	statusBar *StatusBarView

	// Layout containers
	hsplit *pot.Split

	// Focus management
	focus *FocusManager
}

// NewMainView creates a new main layout with the given widgets.
func NewMainView(fileList *FilesListView, diffView *DiffView, statusBar *StatusBarView) *MainView {
	m := &MainView{
		BaseView:  pot.NewBaseView(),
		fileList:  fileList,
		diffView:  diffView,
		statusBar: statusBar,
	}
	m.SetFocusable(false)

	// Set up parent relationships for dirty propagation
	fileList.SetParent(m)
	diffView.SetParent(m)
	statusBar.SetParent(m)

	// Initialize focus manager with focusable children (fileList, diffView)
	// Note: statusBar is not focusable
	m.focus = NewFocusManager(fileList, diffView)

	// Create placeholder widgets for the split
	// (actual rendering is done directly to avoid double-buffering)
	m.hsplit = pot.NewHSplit(nil, nil, 0.3)

	return m
}

// SetBounds sets the layout bounds and propagates to children.
func (m *MainView) SetBounds(bounds pot.Rect) {
	m.BaseView.SetBounds(bounds)

	// StatusBar takes 1 row at the bottom
	statusBarHeight := 1
	contentHeight := bounds.Height - statusBarHeight
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Calculate split widths (30% for file list, 70% for diff view)
	// File list is indented by 1 space on the left
	fileListIndent := 1
	splitRatio := 0.3
	leftWidth := int(float64(bounds.Width) * splitRatio)
	rightWidth := bounds.Width - leftWidth - 1 // -1 for separator

	// Set file list bounds (indented by 1 space)
	m.fileList.SetBounds(pot.NewRect(bounds.X+fileListIndent, bounds.Y, leftWidth-fileListIndent, contentHeight))

	// Set diff view bounds
	m.diffView.SetBounds(pot.NewRect(bounds.X+leftWidth+1, bounds.Y, rightWidth, contentHeight))

	// Set status bar bounds
	m.statusBar.SetBounds(pot.NewRect(bounds.X, bounds.Y+contentHeight, bounds.Width, statusBarHeight))
}

// Render renders the main layout to the buffer.
func (m *MainView) Render(buf *pot.SubBuffer) {
	bounds := m.Bounds()
	statusBarHeight := 1
	contentHeight := bounds.Height - statusBarHeight
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Calculate split widths
	// File list is indented by 1 space on the left
	fileListIndent := 1
	splitRatio := 0.3
	leftWidth := int(float64(bounds.Width) * splitRatio)
	rightWidth := bounds.Width - leftWidth - 1

	filteredCount, totalCount := m.fileList.GetFilterInfo()
	logger.Info("MainView.Render: bounds=%dx%d, buf=%dx%d, left=%d, right=%d, content=%d, files=%d/%d",
		bounds.Width, bounds.Height, buf.Width(), buf.Height(), leftWidth, rightWidth, contentHeight, filteredCount, totalCount)

	// Render file list (indented by 1 space)
	fileListWidth := leftWidth - fileListIndent
	if fileListWidth > 0 && contentHeight > 0 {
		fileListBuf := buf.Sub(pot.NewRect(fileListIndent, 0, fileListWidth, contentHeight))
		m.fileList.Render(fileListBuf)
		logger.Info("MainView.Render: fileList rendered")
	}

	// Render separator
	separatorX := leftWidth
	for y := 0; y < contentHeight; y++ {
		buf.SetString(separatorX, y, "│", lipgloss.NewStyle())
	}

	// Render diff view
	if rightWidth > 0 && contentHeight > 0 {
		diffBuf := buf.Sub(pot.NewRect(leftWidth+1, 0, rightWidth, contentHeight))
		m.diffView.Render(diffBuf)
		logger.Info("MainView.Render: diffView rendered")
	}

	// Render status bar
	if bounds.Width > 0 && statusBarHeight > 0 {
		statusBuf := buf.Sub(pot.NewRect(0, contentHeight, bounds.Width, statusBarHeight))
		m.statusBar.Render(statusBuf)
		logger.Info("MainView.Render: statusBar rendered")
	}
	logger.Info("MainView.Render: complete")
}

// Children returns the child widgets for focus traversal.
func (m *MainView) Children() []pot.View {
	return []pot.View{m.fileList, m.diffView, m.statusBar}
}

// FileList returns the file list widget.
func (m *MainView) FileList() *FilesListView {
	return m.fileList
}

// DiffView returns the diff view model.
func (m *MainView) DiffView() *DiffView {
	return m.diffView
}

// StatusBar returns the status bar widget.
func (m *MainView) StatusBar() *StatusBarView {
	return m.statusBar
}

// GetFocusedPane returns which pane is currently focused.
func (m *MainView) GetFocusedPane() Pane {
	return m.focus.GetFocusedPane()
}

// SetFocusedPane sets which pane is focused.
func (m *MainView) SetFocusedPane(pane Pane) {
	m.focus.SetFocusedPane(pane)
}

// FocusNext moves focus to the next focusable child.
// Returns true if focus was moved, false if already at the last child.
func (m *MainView) FocusNext() bool {
	return m.focus.FocusNext()
}

// FocusPrev moves focus to the previous focusable child.
// Returns true if focus was moved, false if already at the first child.
func (m *MainView) FocusPrev() bool {
	return m.focus.FocusPrev()
}
