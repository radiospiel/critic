package app

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/logger"
	"git.15b.it/eno/critic/internal/ui"
	ctypes "git.15b.it/eno/critic/pkg/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the main application model
type Model struct {
	fileList            ui.FileListModel
	diffView            ui.DiffViewModel
	layout              ui.LayoutModel
	diff                *ctypes.Diff
	paths               []string
	watcher             *git.Watcher
	err                 error
	width               int
	height              int
	ready               bool
	highlightingEnabled bool
	reloading           bool
	diffMode            git.DiffMode
}

// NewModel creates a new application model
func NewModel(paths []string, highlightingEnabled bool) Model {
	logger.Info("NewModel: Creating model with %d paths, highlighting=%v", len(paths), highlightingEnabled)
	diffView := ui.NewDiffViewModel()
	diffView.SetHighlightingEnabled(highlightingEnabled)

	// Initialize file watcher
	watcher, err := git.NewWatcher(100) // 100ms debounce
	if err != nil {
		logger.Error("NewModel: Failed to create watcher: %v", err)
		watcher = nil // Continue without watcher if it fails
	} else {
		logger.Info("NewModel: Watcher created successfully")
	}

	fileList := ui.NewFileListModel()
	fileList.SetFocused(true) // Start with file list focused

	return Model{
		fileList:            fileList,
		diffView:            diffView,
		layout:              ui.NewLayoutModel(),
		paths:               paths,
		watcher:             watcher,
		highlightingEnabled: highlightingEnabled,
		diffMode:            git.DiffToMergeBase, // Default mode
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	logger.Info("Init: Starting application initialization")
	cmds := []tea.Cmd{
		loadDiffCmd(m.paths, m.diffMode),
		tea.EnterAltScreen,
	}

	// Start file watcher if available
	if m.watcher != nil {
		logger.Info("Init: Starting file watcher")
		if err := m.watcher.WatchPaths(m.paths); err == nil {
			logger.Info("Init: WatchPaths succeeded, starting waitForFileChanges")
			cmds = append(cmds, waitForFileChanges(m.watcher))
		} else {
			logger.Error("Init: WatchPaths failed: %v", err)
		}
	} else {
		logger.Info("Init: No watcher available")
	}

	return tea.Batch(cmds...)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab", "shift+tab":
			m.layout.ToggleFocus()
			// Update focus state on both panes
			m.fileList.SetFocused(m.layout.GetFocusedPane() == ui.FileListPane)
			m.diffView.SetFocused(m.layout.GetFocusedPane() == ui.DiffViewPane)

		case "m":
			// Cycle through diff modes
			m.diffMode = m.nextDiffMode()
			logger.Info("Update: Switching to diff mode: %s", m.diffMode.String())
			return m, loadDiffCmd(m.paths, m.diffMode)

		case " ": // Space - page down diff view regardless of focus
			// Create a synthetic pgdown key message
			pgDownMsg := tea.KeyMsg{Type: tea.KeyType(tea.KeyPgDown)}
			cmd := m.diffView.Update(pgDownMsg)
			cmds = append(cmds, cmd)

		case "shift+ ": // Shift+Space - page up diff view regardless of focus
			// Create a synthetic pgup key message
			pgUpMsg := tea.KeyMsg{Type: tea.KeyType(tea.KeyPgUp)}
			cmd := m.diffView.Update(pgUpMsg)
			cmds = append(cmds, cmd)

		case "?":
			// TODO: Show help screen
			return m, nil

		default:
			// Route key messages to focused pane
			if m.layout.GetFocusedPane() == ui.FileListPane {
				newFileList, cmd := m.fileList.Update(msg)
				m.fileList = newFileList
				// Update diff view when file selection changes
				setFileCmd := m.diffView.SetFile(m.fileList.GetActiveFile())
				cmds = append(cmds, cmd, setFileCmd)
			} else {
				cmd := m.diffView.Update(msg)
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

		// Re-render diff view on window resize (to recalculate line wrapping)
		if m.diffView.GetFile() != nil {
			cmd := m.diffView.SetFile(m.diffView.GetFile())
			cmds = append(cmds, cmd)
		}

	case diffLoadedMsg:
		logger.Info("Update: Received diffLoadedMsg")
		m.diff = msg.diff
		m.err = msg.err
		if m.diff != nil {
			logger.Info("Update: Diff loaded with %d files", len(m.diff.Files))
			// Remember currently selected file path
			var currentPath string
			if activeFile := m.fileList.GetActiveFile(); activeFile != nil {
				currentPath = activeFile.NewPath
				if currentPath == "" {
					currentPath = activeFile.OldPath
				}
			}

			// Update file list with new diff
			m.fileList.SetFiles(m.diff.Files)

			// Try to restore selection to the same file
			if currentPath != "" {
				logger.Info("Update: Restoring selection to %s", currentPath)
				m.fileList.SelectByPath(currentPath)
			}

			// Update diff view with (potentially) new content
			cmd := m.diffView.SetFile(m.fileList.GetActiveFile())
			cmds = append(cmds, cmd)
		} else if m.err != nil {
			logger.Error("Update: Diff loading failed: %v", m.err)
		}

	case fileChangedMsg:
		// Reload diff when files change
		logger.Info("Update: Received fileChangedMsg, reloading diff")
		m.reloading = true
		return m, tea.Batch(
			loadDiffCmd(m.paths, m.diffMode),
			waitForFileChanges(m.watcher),
		)

	default:
		// Route other messages to diff view (like diffRenderedMsg)
		cmd := m.diffView.Update(msg)
		cmds = append(cmds, cmd)
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

// nextDiffMode cycles to the next diff mode
func (m Model) nextDiffMode() git.DiffMode {
	switch m.diffMode {
	case git.DiffToMergeBase:
		return git.DiffToLastCommit
	case git.DiffToLastCommit:
		return git.DiffUnstaged
	case git.DiffUnstaged:
		return git.DiffToMergeBase
	default:
		return git.DiffToMergeBase
	}
}

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	var parts []string

	// Show current diff mode
	parts = append(parts, fmt.Sprintf("Mode: %s", m.diffMode.String()))

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
	parts = append(parts, "m: mode • Tab: switch • ?: help • q: quit")

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

func loadDiffCmd(paths []string, mode git.DiffMode) tea.Cmd {
	return func() tea.Msg {
		diff, err := git.GetDiff(paths, mode)
		return diffLoadedMsg{diff: diff, err: err}
	}
}

func waitForFileChanges(watcher *git.Watcher) tea.Cmd {
	if watcher == nil {
		logger.Info("waitForFileChanges: No watcher, returning nil")
		return nil
	}

	return func() tea.Msg {
		logger.Info("waitForFileChanges: Waiting for file changes...")
		<-watcher.Changes()
		logger.Info("waitForFileChanges: File change received, returning fileChangedMsg")
		return fileChangedMsg{}
	}
}
