package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"git.15b.it/eno/critic/internal/cli"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/logger"
	"git.15b.it/eno/critic/internal/ui"
	ctypes "git.15b.it/eno/critic/pkg/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the main application model
type Model struct {
	fileList     ui.FileListModel
	diffView     ui.DiffViewModel
	layout       ui.LayoutModel
	diff         *ctypes.Diff
	bases        []string        // List of base refs
	current      string          // Current target ref
	currentBase  int             // Index of current base
	paths        []string        // Paths to diff
	extensions   []string        // File extensions to include
	resolver     *git.BaseResolver // Base resolver with polling
	watcher      *git.Watcher
	err          error
	width        int
	height       int
	ready        bool
	reloading    bool
}

// NewModel creates a new application model
func NewModel(args *cli.Args) Model {
	logger.Info("NewModel: Creating model with %d paths, %d bases", len(args.Paths), len(args.Bases))
	diffView := ui.NewDiffViewModel()
	diffView.SetHighlightingEnabled(true) // Always enable highlighting

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
		fileList:   fileList,
		diffView:   diffView,
		layout:     ui.NewLayoutModel(),
		bases:      args.Bases,
		current:    args.Current,
		currentBase: 0, // Start with first base
		paths:      args.Paths,
		extensions: args.Extensions,
		watcher:    watcher,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	logger.Info("Init: Starting application initialization")

	// Check terminal color support
	checkTerminalColors()

	cmds := []tea.Cmd{
		initBaseResolverCmd(&m),
		loadDiffCmd(&m),
		disableTerminalLineWrap, // This now handles alternate screen + nowrap
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
			return m, tea.Sequence(enableTerminalLineWrap, tea.Quit)

		case "tab", "shift+tab":
			m.layout.ToggleFocus()
			// Update focus state on both panes
			m.fileList.SetFocused(m.layout.GetFocusedPane() == ui.FileListPane)
			m.diffView.SetFocused(m.layout.GetFocusedPane() == ui.DiffViewPane)

		case "b":
			// Cycle through bases
			m.currentBase = (m.currentBase + 1) % len(m.bases)
			logger.Info("Update: Switching to base %d: %s", m.currentBase, m.bases[m.currentBase])
			return m, loadDiffCmd(&m)

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

		// Terminal.app handles resize cleanly with fullscreen+nowrap modes,
		// but iTerm2 and others may need explicit repaint to avoid artifacts
		return m, tea.ClearScreen

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
		// Check if the changed file is currently being viewed
		activeFile := m.fileList.GetActiveFile()
		if activeFile != nil {
			// Get the git-relative path of the currently viewed file
			currentGitPath := activeFile.NewPath
			if currentGitPath == "" {
				currentGitPath = activeFile.OldPath
			}

			// Convert watcher absolute path to git-relative path
			changedGitPath := git.AbsPathToGitPath(msg.path)

			logger.Info("Update: File changed: watcher=%q -> git=%q, active=%q", msg.path, changedGitPath, currentGitPath)

			// Compare git-relative paths
			if changedGitPath != "" && changedGitPath == currentGitPath {
				logger.Info("Update: MATCH! Changed file is currently viewed, immediately re-rendering")
				// Immediately re-render the current file
				cmd := m.diffView.SetFile(activeFile)
				cmds = append(cmds, cmd)
			} else {
				logger.Info("Update: No match - changed file is not currently viewed")
			}
		}

		// Also reload the full diff in the background to update file list
		logger.Info("Update: Reloading full diff in background")
		m.reloading = true
		return m, tea.Batch(
			append(cmds,
				loadDiffCmd(&m),
				waitForFileChanges(m.watcher),
			)...,
		)

	case baseResolverInitializedMsg:
		logger.Info("Update: BaseResolver initialized")
		m.resolver = msg.resolver
		return m, nil

	case baseChangedMsg:
		// Base changed due to polling - reload diff
		logger.Info("Update: Received baseChangedMsg, reloading diff")
		return m, loadDiffCmd(&m)

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

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	var parts []string

	// Show current base and target
	if len(m.bases) > 0 {
		base := m.bases[m.currentBase]
		parts = append(parts, fmt.Sprintf("[B]ase: %s → %s", base, m.current))
	}

	// Show file count
	if m.diff != nil {
		parts = append(parts, fmt.Sprintf("Files: %d", len(m.diff.Files)))
	}

	// Show help hint
	parts = append(parts, "[Tab] switch • [?] help • [q] quit")

	status := strings.Join(parts, " • ")

	// Truncate if too long (account for padding)
	maxLen := m.width - 4 // subtract padding and some margin
	if maxLen > 3 && len(status) > maxLen {
		status = status[:maxLen-3] + "..."
	}

	return ui.GetStatusStyle().
		Width(m.width).
		MaxWidth(m.width).
		Inline(true).
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

type fileChangedMsg struct {
	path string
}

type baseResolverInitializedMsg struct {
	resolver *git.BaseResolver
}

type baseChangedMsg struct{}

// checkTerminalColors verifies the terminal supports at least 256 colors
func checkTerminalColors() {
	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "" {
		fmt.Fprintf(os.Stderr, "Warning: TERM environment variable not set\n")
		fmt.Fprintf(os.Stderr, "Syntax highlighting may not work correctly\n")
		return
	}

	// Try to get color count using tput
	cmd := exec.Command("tput", "colors")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Could not determine color support: %v", err)
		return
	}

	colors := strings.TrimSpace(string(output))
	colorCount := 0
	fmt.Sscanf(colors, "%d", &colorCount)

	if colorCount < 256 {
		fmt.Fprintf(os.Stderr, "Warning: Terminal supports only %d colors (256+ recommended)\n", colorCount)
		fmt.Fprintf(os.Stderr, "TERM=%s - consider using xterm-256color\n", term)
		fmt.Fprintf(os.Stderr, "Syntax highlighting backgrounds may not display correctly\n\n")
	} else {
		logger.Info("Terminal color support: %d colors", colorCount)
	}
}

// Commands

// disableTerminalLineWrap sends escape sequence to disable line wrapping
func disableTerminalLineWrap() tea.Msg {
	fmt.Print("\x1b[?7l")    // DECAWM - disable auto wrap mode
	fmt.Print("\x1b[?1049h") // Use alternate screen buffer with better isolation
	return nil
}

// enableTerminalLineWrap sends escape sequence to re-enable line wrapping
func enableTerminalLineWrap() tea.Msg {
	fmt.Print("\x1b[?1049l") // Exit alternate screen buffer
	fmt.Print("\x1b[?7h")    // DECAWM - enable auto wrap mode
	return nil
}

func initBaseResolverCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		resolver, err := git.NewBaseResolver(m.bases, m.current, func() {
			// This callback is called when bases change
			// We'll send a message to trigger reload
			logger.Info("BaseResolver: Bases changed, triggering reload")
		})
		if err != nil {
			logger.Error("Failed to initialize base resolver: %v", err)
			return nil
		}
		return baseResolverInitializedMsg{resolver: resolver}
	}
}

func loadDiffCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		// Get current base name
		baseName := m.bases[m.currentBase]

		// Resolve base to commit SHA
		var baseCommit string
		if m.resolver != nil {
			sha, ok := m.resolver.GetResolvedBase(baseName)
			if !ok {
				// Fall back to resolving directly
				resolvedSHA, err := resolveBase(baseName)
				if err != nil {
					return diffLoadedMsg{diff: nil, err: fmt.Errorf("failed to resolve base %s: %w", baseName, err)}
				}
				baseCommit = resolvedSHA
			} else {
				baseCommit = sha
			}
		} else {
			// No resolver yet, resolve directly
			resolvedSHA, err := resolveBase(baseName)
			if err != nil {
				return diffLoadedMsg{diff: nil, err: fmt.Errorf("failed to resolve base %s: %w", baseName, err)}
			}
			baseCommit = resolvedSHA
		}

		// Resolve target (might be "current" or a git ref)
		var targetCommit string
		if m.current == "current" {
			targetCommit = "current"
		} else {
			// Resolve target ref to commit SHA
			sha, err := git.ResolveRef(m.current)
			if err != nil {
				return diffLoadedMsg{diff: nil, err: fmt.Errorf("failed to resolve target %s: %w", m.current, err)}
			}
			targetCommit = sha
		}

		logger.Info("loadDiffCmd: Loading diff from %s (%s) to %s (%s)", baseName, baseCommit, m.current, targetCommit)

		// Use GetDiffBetween to get diff between specific commits
		diff, err := git.GetDiffBetween(baseCommit, targetCommit, m.paths)

		return diffLoadedMsg{diff: diff, err: err}
	}
}

func resolveBase(base string) (string, error) {
	// Check if base is already a commit SHA (hexadecimal)
	// If it is, use it directly without computing merge-base
	if git.IsCommitSHA(base) {
		// Verify it's a valid ref and return the full SHA
		return git.ResolveRef(base)
	}

	// For branch names (not commit SHAs), compute merge-base with HEAD
	// This ensures we diff from where the branch diverged, not the branch tip
	baseSHA, err := git.ResolveRef(base)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref %s: %w", base, err)
	}

	// Get merge-base between HEAD and the branch
	mergeBase, err := git.GetMergeBaseBetween("HEAD", baseSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get merge base with %s: %w", base, err)
	}

	return mergeBase, nil
}

func waitForFileChanges(watcher *git.Watcher) tea.Cmd {
	if watcher == nil {
		logger.Info("waitForFileChanges: No watcher, returning nil")
		return nil
	}

	return func() tea.Msg {
		logger.Info("waitForFileChanges: Waiting for file changes...")
		change := <-watcher.Changes()
		logger.Info("waitForFileChanges: File change received for %s, returning fileChangedMsg", change.Path)
		return fileChangedMsg{path: change.Path}
	}
}
