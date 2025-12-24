package app

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/ui"
	ctypes "git.15b.it/eno/critic/pkg/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the main application model
type Model struct {
	fileList ui.FileListModel
	diffView ui.DiffViewModel
	layout   ui.LayoutModel
	diff     *ctypes.Diff
	paths    []string
	watcher  *git.Watcher
	err      error
	width    int
	height   int
	ready    bool
}

// NewModel creates a new application model
func NewModel(paths []string) Model {
	return Model{
		fileList: ui.NewFileListModel(),
		diffView: ui.NewDiffViewModel(),
		layout:   ui.NewLayoutModel(),
		paths:    paths,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	// Start file watcher
	watcher, err := git.NewWatcher(300) // 300ms debounce
	if err == nil {
		m.watcher = watcher
		if err := watcher.WatchPaths(m.paths); err == nil {
			return tea.Batch(
				loadDiffCmd(m.paths),
				waitForFileChanges(watcher),
				tea.EnterAltScreen,
			)
		}
	}

	return tea.Batch(
		loadDiffCmd(m.paths),
		tea.EnterAltScreen,
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			m.layout.ToggleFocus()

		case "?":
			// TODO: Show help screen
			return m, nil

		default:
			// Route key messages to focused pane
			if m.layout.GetFocusedPane() == ui.FileListPane {
				newFileList, cmd := m.fileList.Update(msg)
				m.fileList = newFileList
				// Update diff view when file selection changes
				m.diffView.SetFile(m.fileList.GetActiveFile())
				cmds = append(cmds, cmd)
			} else {
				newDiffView, cmd := m.diffView.Update(msg)
				m.diffView = newDiffView
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout.SetSize(msg.Width, msg.Height)

		// Update component sizes
		leftWidth, leftHeight := m.layout.GetFileListSize()
		m.fileList.SetSize(leftWidth, leftHeight)

		rightWidth, rightHeight := m.layout.GetDiffViewSize()
		m.diffView.SetSize(rightWidth, rightHeight)

		if !m.ready {
			m.ready = true
		}

	case diffLoadedMsg:
		m.diff = msg.diff
		m.err = msg.err
		if m.diff != nil {
			m.fileList.SetFiles(m.diff.Files)
			// Set initial file in diff view
			m.diffView.SetFile(m.fileList.GetActiveFile())
		}

	case fileChangedMsg:
		// Reload diff when files change
		return m, tea.Batch(
			loadDiffCmd(m.paths),
			waitForFileChanges(m.watcher),
		)
	}

	return m, tea.Batch(cmds...)
}

// View renders the application
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.err != nil {
		return renderError(m.err)
	}

	// Render panes
	fileListView := m.fileList.View()
	diffViewView := m.diffView.View()

	// Render layout
	mainView := m.layout.RenderSplitView(fileListView, diffViewView)

	// Render status bar
	statusBar := m.renderStatusBar()

	// Combine main view and status bar
	return lipgloss.JoinVertical(lipgloss.Left, mainView, statusBar)
}

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	var parts []string

	// Show current branch
	branch, err := git.GetCurrentBranch()
	if err == nil {
		parts = append(parts, fmt.Sprintf("Branch: %s", branch))
	}

	// Show file count
	if m.diff != nil {
		parts = append(parts, fmt.Sprintf("Files: %d", len(m.diff.Files)))
	}

	// Show help hint
	parts = append(parts, "Tab: switch pane • ?: help • q: quit")

	status := strings.Join(parts, " • ")

	return lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#666"}).
		Padding(0, 1).
		Render(status)
}

// renderError renders an error message
func renderError(err error) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")). // Red
		Bold(true).
		Padding(1, 2).
		Render(fmt.Sprintf("Error: %s", err))
}

// Messages

type diffLoadedMsg struct {
	diff *ctypes.Diff
	err  error
}

type fileChangedMsg struct{}

// Commands

func loadDiffCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		diff, err := git.GetDiff(paths)
		return diffLoadedMsg{diff: diff, err: err}
	}
}

func waitForFileChanges(watcher *git.Watcher) tea.Cmd {
	if watcher == nil {
		return nil
	}

	return func() tea.Msg {
		<-watcher.Changes()
		return fileChangedMsg{}
	}
}
