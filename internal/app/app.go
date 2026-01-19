package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"git.15b.it/eno/critic/internal/config"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/messagedb"
	"git.15b.it/eno/critic/internal/tui"
	"git.15b.it/eno/critic/pkg/critic"
	ctypes "git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/simple-go/logger"
	"git.15b.it/eno/critic/teapot"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FilterMode represents the current file/hunk filter mode
type FilterMode int

const (
	// FilterModeNone shows all files and hunks (default)
	FilterModeNone FilterMode = iota
	// FilterModeWithComments shows only files with comments, and only hunks with comments
	FilterModeWithComments
	// FilterModeWithUnresolved shows only files with unresolved comments, and only hunks with unresolved comments
	FilterModeWithUnresolved
)

// String returns a display name for the filter mode
func (m FilterMode) String() string {
	switch m {
	case FilterModeWithComments:
		return "With Comments"
	case FilterModeWithUnresolved:
		return "Unresolved Only"
	default:
		return "All"
	}
}

// Args represents parsed command-line arguments
type Args struct {
	Bases       []string // List of base points (e.g., ["main", "origin/main", "HEAD"])
	Paths       []string // Paths to diff
	Extensions  []string // File extensions to include
	NoAnimation bool     // Disable animations
}

// GetDefaultBases returns the default base points based on git state
func GetDefaultBases() ([]string, error) {
	bases := []string{}

	// 1. Add main/master if it exists (will use merge-base automatically)
	if branchExists("main") {
		bases = append(bases, "main")
	} else if branchExists("master") {
		bases = append(bases, "master")
	}

	// 2. Add origin/<current-branch> if it exists
	branch, err := git.GetCurrentBranch()
	if err == nil && branch != "" {
		originBranch := "origin/" + branch
		// Check if origin branch exists
		if branchExists(originBranch) {
			bases = append(bases, originBranch)
		}
	}

	// 3. Add HEAD (last committed version)
	bases = append(bases, "HEAD")

	return bases, nil
}

// branchExists checks if a git ref exists
func branchExists(ref string) bool {
	// Try to resolve the ref
	_, err := git.ResolveRef(ref)
	return err == nil
}

// Run runs the application with the given arguments
func Run(args *Args) error {
	logger.Info("=== Critic starting ===")

	// Check if we're in a git repository
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Set default bases if none were specified
	if len(args.Bases) == 0 {
		bases, err := GetDefaultBases()
		if err != nil {
			return fmt.Errorf("failed to determine default bases: %w", err)
		}
		args.Bases = bases
	}

	// Set default extensions if none were specified
	if len(args.Extensions) == 0 {
		args.Extensions = config.DefaultFileExtensions
	}

	// Create and run the application
	m := NewModel(args)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("application error: %w", err)
	}

	return nil
}

// Model represents the main application model
type Model struct {
	fileList      *tui.FileListView
	diffView      *tui.DiffView
	commentEditor tui.CommentEditor
	statusBar     *tui.StatusBarView
	mainLayout    *tui.MainView
	compositor    *teapot.Compositor
	layout        tui.LayoutView // TODO: Remove after full migration
	diff          *ctypes.Diff
	bases         []string          // List of base refs
	currentBase   int               // Index of current base
	paths         []string          // Paths to diff
	extensions    []string          // File extensions to include
	resolver      *git.BaseResolver // Base resolver with polling
	messaging     critic.Messaging  // Messaging interface for conversations
	filterMode    FilterMode        // Current filter mode (None, WithComments, WithUnresolved)
	noAnimation   bool              // Whether animations are disabled
	tickCount     int               // Debug: count of animation ticks
	err           error
	width         int
	height        int
	ready         bool
	showHelp      bool // Whether to show help screen
}

// NewModel creates a new application model
func NewModel(args *Args) Model {
	logger.Info("NewModel: Creating model with %d paths, %d bases", len(args.Paths), len(args.Bases))
	diffView := tui.NewDiffView()
	diffView.SetHighlightingEnabled(true) // Always enable highlighting

	fileList := tui.NewFileListView()
	fileList.SetFocused(true) // Start with file list focused

	// Initialize message database
	gitRoot, err := git.GetGitRoot()
	if err != nil {
		logger.Fatal("Failed to get git root: %v", err)
	}
	mdb, err := messagedb.New(gitRoot)
	if err != nil {
		logger.Fatal("Failed to initialize message database: %v", err)
	}

	diffView.SetMessaging(mdb)
	fileList.SetMessaging(mdb)

	statusBar := tui.NewStatusBarView()
	statusBar.SetFilter("All") // Default filter mode

	// Subscribe statusbar to receive tick notifications for clock updates
	teapot.SubscribeToGlobalTicks(statusBar)

	// Create the main layout (VBox with HSplit and StatusBar)
	mainLayout := tui.NewMainView(fileList, diffView, statusBar)

	// Create compositor with main layout as root
	compositor := teapot.NewCompositor(mainLayout)

	return Model{
		fileList:      fileList,
		diffView:      diffView,
		commentEditor: tui.NewCommentEditor(),
		statusBar:     statusBar,
		mainLayout:    mainLayout,
		compositor:    compositor,
		layout:        tui.NewLayoutView(),
		bases:         args.Bases,
		currentBase:   0, // Start with first base
		paths:         args.Paths,
		extensions:    args.Extensions,
		messaging:     mdb,
		noAnimation:   args.NoAnimation,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	logger.Info("Init: Starting application initialization")

	// Enable compositor debug logging
	teapot.CompositorDebug = true

	// Check terminal color support
	checkTerminalColors()

	cmds := []tea.Cmd{
		initBaseResolverCmd(&m),
		loadDiffCmd(&m),
		disableTerminalLineWrap, // This now handles alternate screen + nowrap
		// Always start compositor tick loop
		tea.Tick(teapot.ComposerTickInterval, func(_ time.Time) tea.Msg {
			return teapot.ComposerTickMsg{}
		}),
	}

	return tea.Batch(cmds...)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If comment editor is active, route all keys to it
		if m.commentEditor.IsActive() {
			cmd := m.commentEditor.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Sequence(enableTerminalLineWrap, tea.Quit)

		case "tab", "shift+tab":
			m.layout.ToggleFocus()
			// Update focus state on both panes
			m.fileList.SetFocused(m.layout.GetFocusedPane() == tui.FileListPane)
			m.diffView.SetFocused(m.layout.GetFocusedPane() == tui.DiffViewPane)

		case "b":
			// Cycle through bases
			m.currentBase = (m.currentBase + 1) % len(m.bases)
			logger.Info("Update: Switching to base %d: %s", m.currentBase, m.bases[m.currentBase])
			return m, loadDiffCmd(&m)

		case "f", "F":
			// Cycle through filter modes (works from both file list and diff pane)
			m.filterMode = (m.filterMode + 1) % 3
			logger.Info("Update: Switching to filter mode %d: %s", m.filterMode, m.filterMode.String())
			// Update status bar
			m.statusBar.SetFilter(m.filterMode.String())
			// Re-apply filter to file list and update diff view
			m.applyFilterMode()
			cmd := m.diffView.SetFile(m.fileList.GetActiveFile())
			cmds = append(cmds, cmd)
			m.mainLayout.Repaint() // Trigger widget re-render
			return m, tea.Batch(cmds...)

		case " ": // Space - page down diff view regardless of focus
			// Scroll by height - 3 (but at least 1 row) and position cursor on second line
			cmd := m.diffView.ScrollPageDown()
			cmds = append(cmds, cmd)
			m.mainLayout.Repaint() // Trigger widget re-render

		case "shift+ ": // Shift+Space - page up diff view regardless of focus
			// Scroll by height - 3 (but at least 1 row) and position cursor on second line
			cmd := m.diffView.ScrollPageUp()
			cmds = append(cmds, cmd)
			m.mainLayout.Repaint() // Trigger widget re-render

		case "?":
			// Toggle help screen
			m.showHelp = !m.showHelp
			return m, nil

		case "enter":
			// Activate comment editor when focused on diff view
			if m.layout.GetFocusedPane() == tui.DiffViewPane {
				activeFile := m.fileList.GetActiveFile()
				if activeFile != nil {
					// Get the current cursor line from diff view
					cursorLine := m.diffView.GetCursorLine()

					// Check if the cursor is on a comment line
					isCommentLine, sourceLine := m.diffView.IsCommentLine(cursorLine)

					// Determine the source line number to use
					var lineNum int
					var conv *critic.Conversation

					if isCommentLine {
						// Cursor is on a comment preview line - reply to that conversation
						lineNum = sourceLine
						// Load the full conversation from the messaging interface
						uuid := m.diffView.GetConversationUUIDAtLine(cursorLine)
						if uuid != "" {
							if c, err := m.messaging.GetFullConversation(uuid); err == nil {
								conv = c
							}
						}
					} else {
						// Cursor is on a regular diff line - get the source line number
						lineNum = m.diffView.GetSourceLine(cursorLine)
						if lineNum == 0 {
							// Can't comment on this line (e.g., header line)
							return m, nil
						}
						// No existing conversation - this will be a new comment
						conv = nil
					}

					cmd := m.commentEditor.ActivateWithConversation(lineNum, conv)
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				}
			}

		default:
			// Route key messages to focused pane
			if m.layout.GetFocusedPane() == tui.FileListPane {
				prevFile := m.fileList.GetActiveFile()
				_, cmd := m.fileList.HandleKey(msg)
				// Update diff view when file selection changes
				if m.fileList.GetActiveFile() != prevFile {
					newFile := m.fileList.GetActiveFile()
					if newFile != nil {
						logger.Info("File selected: %s", newFile.NewPath)
					}
					setFileCmd := m.diffView.SetFile(newFile)
					cmds = append(cmds, cmd, setFileCmd)
					m.mainLayout.Repaint() // Trigger widget re-render
				} else {
					cmds = append(cmds, cmd)
					m.mainLayout.Repaint() // Trigger widget re-render
				}
			} else {
				cmd := m.diffView.Update(msg)
				cmds = append(cmds, cmd)
				m.mainLayout.Repaint() // Trigger widget re-render after cursor movement
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update compositor and main layout size
		m.compositor.Resize(msg.Width, msg.Height)

		// Keep legacy layout for now (can be removed after full migration)
		m.layout.SetSize(msg.Width, msg.Height)

		// Update comment editor size (80% width, centered)
		editorWidth := msg.Width * 80 / 100
		editorHeight := msg.Height - 6 // Leave some vertical padding
		m.commentEditor.SetSize(editorWidth, editorHeight)

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

			// Apply filter mode (this handles file selection and filtering)
			m.applyFilterMode()

			// Update status bar
			if len(m.bases) > 0 {
				m.statusBar.SetBase(m.bases[m.currentBase])
			}
			m.statusBar.SetFilter(m.filterMode.String())
			stats := computeDiffStats(m.diff)
			m.statusBar.SetDiffStats(stats.Added, stats.Deleted, stats.Moved)

			// Update diff view with (potentially) new content
			cmd := m.diffView.SetFile(m.fileList.GetActiveFile())
			cmds = append(cmds, cmd)

			// Mark layout dirty since diffView changed (diffView is a model, not a widget)
			m.mainLayout.Repaint()
		} else if m.err != nil {
			logger.Error("Update: Diff loading failed: %v", m.err)
		}

	case baseResolverInitializedMsg:
		logger.Info("Update: BaseResolver initialized")
		m.resolver = msg.resolver
		return m, nil

	case baseChangedMsg:
		// Base changed due to polling - reload diff
		logger.Info("Update: Received baseChangedMsg, reloading diff")
		return m, loadDiffCmd(&m)

	case tui.RequestNextFileMsg:
		// User scrolled past end of diff, load next file
		if m.fileList.SelectNext() {
			cmd := m.diffView.SetFile(m.fileList.GetActiveFile())
			cmds = append(cmds, cmd)
		}

	case tui.RequestPrevFileMsg:
		// User scrolled before start of diff, load previous file and go to bottom
		if m.fileList.SelectPrev() {
			m.diffView.SetGotoBottomOnLoad()
			cmd := m.diffView.SetFile(m.fileList.GetActiveFile())
			cmds = append(cmds, cmd)
		}

	case tui.CommentSavedMsg:
		// Save the comment using the messaging interface
		activeFile := m.fileList.GetActiveFile()
		if activeFile != nil && msg.Comment != "" {
			filePath := activeFile.NewPath
			if filePath == "" {
				filePath = activeFile.OldPath
			}

			// Get current git commit for code_version
			codeVersion := m.getCurrentCodeVersion()

			// Get the cursor line to check for existing conversation
			cursorLine := m.diffView.GetCursorLine()
			existingUUID := m.diffView.GetConversationUUIDAtLine(cursorLine)

			if existingUUID != "" {
				// There's an existing conversation - add a reply
				replyMsg, err := m.messaging.ReplyToConversation(
					existingUUID,
					msg.Comment,
					critic.AuthorHuman,
				)
				if err != nil {
					logger.Error("Failed to create reply: %v", err)
					return m, nil
				}
				logger.Info("Created reply %s to conversation %s", replyMsg.UUID, existingUUID)
			} else {
				// No existing conversation - create a new one
				// Get context around the line being commented
				context := git.GetLineContext(filePath, msg.LineNum, codeVersion)
				conversation, err := m.messaging.CreateConversation(
					critic.AuthorHuman,
					msg.Comment,
					filePath,
					msg.LineNum,
					codeVersion,
					context,
				)
				if err != nil {
					logger.Error("Failed to create conversation: %v", err)
					return m, nil
				}
				logger.Info("Created conversation %s at %s:%d", conversation.UUID, filePath, msg.LineNum)
			}

			// Force refresh the diff view to show the new/updated comment inline
			cmd := m.diffView.RefreshFile()
			cmds = append(cmds, cmd)
		}

	case teapot.ComposerTickMsg:
		// Notify all tick subscribers (e.g., statusbar clock)
		teapot.NotifyGlobalTickSubscribers()

		// Continue ticking for animations
		cmds = append(cmds, tea.Tick(teapot.ComposerTickInterval, func(_ time.Time) tea.Msg {
			return teapot.ComposerTickMsg{}
		}))

	default:
		// Route other messages to diff view (like diffRenderedMsg)
		cmd := m.diffView.Update(msg)
		cmds = append(cmds, cmd)

		// Also route to comment editor in case it's a message it handles
		editorCmd := m.commentEditor.Update(msg)
		cmds = append(cmds, editorCmd)

		// Trigger repaint in case the message updated widget state (e.g., diffRenderedMsg)
		m.mainLayout.Repaint()
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

	// Use compositor for widget-based rendering
	view := m.compositor.Render()

	// Debug: log compositor output
	w, h := m.compositor.Size()
	logger.Info("View: compositor size=%dx%d, view len=%d, lines=%d", w, h, len(view), len(strings.Split(view, "\n")))

	// Overlay comment editor if active
	if m.commentEditor.IsActive() {
		// Render comment editor to buffer
		editorBuf := teapot.NewBuffer(m.commentEditor.Width(), m.commentEditor.Height())
		editorSub := editorBuf.Sub(editorBuf.Bounds())
		m.commentEditor.Render(editorSub)
		editorView := editorBuf.RenderToString()
		editorLines := strings.Split(editorView, "\n")

		// Calculate horizontal padding for centering (80% width means 10% padding on each side)
		leftPadding := m.width * 10 / 100

		// Calculate vertical position for centering
		lines := strings.Split(view, "\n")
		startLine := (len(lines) - len(editorLines)) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Overlay editor on view with horizontal centering
		for i, editorLine := range editorLines {
			lineIdx := startLine + i
			if lineIdx < len(lines) {
				// Pad the editor line to center it
				paddedLine := strings.Repeat(" ", leftPadding) + editorLine
				// Ensure the line fills the width
				if len([]rune(paddedLine)) < m.width {
					paddedLine += strings.Repeat(" ", m.width-len([]rune(paddedLine)))
				}
				lines[lineIdx] = paddedLine
			}
		}
		return strings.Join(lines, "\n")
	}

	// Overlay help screen if showing
	if m.showHelp {
		return m.renderHelpOverlay(view)
	}

	return view
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
		resolver, err := git.NewBaseResolver(m.bases, "HEAD", func() {
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

		// Always diff against HEAD
		targetCommit, err := git.ResolveRef("HEAD")
		if err != nil {
			return diffLoadedMsg{diff: nil, err: fmt.Errorf("failed to resolve HEAD: %w", err)}
		}

		logger.Info("loadDiffCmd: Loading diff from %s (%s) to HEAD (%s)", baseName, baseCommit, targetCommit)

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

// renderHelpOverlay renders the help screen overlay
func (m Model) renderHelpOverlay(underlay string) string {
	helpContent := `
  CRITIC - Git Diff Viewer

  NAVIGATION
    Tab           Switch focus between file list and diff view
    ↑/↓           Navigate up/down
    Shift+↑/↓     Move 10 lines up/down
    PgUp/PgDn     Page up/down in diff view
    Space         Page down in diff view
    Shift+Space   Page up in diff view
    Home/End      Jump to start/end
    j/k           Vim-style navigation (down/up)

  FILE LIST
    Enter         Select and view file
    g             Jump to top
    G             Jump to bottom

  DIFF VIEW
    [/]           Previous/next hunk
    n/p           Next/previous file

  BASE SWITCHING
    b/B           Switch to next/previous base

  FILTERING
    f/F           Cycle filter mode (All → With Comments → Unresolved Only)

  OTHER
    r             Reload diff
    ?             Toggle this help screen
    q             Quit

  Press ? to close this help
`

	// Create a centered box with the help content
	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(60).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252"))

	helpBox := helpStyle.Render(helpContent)

	// Center the help box on the screen
	verticalPadding := (m.height - lipgloss.Height(helpBox)) / 2
	horizontalPadding := (m.width - lipgloss.Width(helpBox)) / 2

	if verticalPadding < 0 {
		verticalPadding = 0
	}
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	// Position the help box
	positioned := lipgloss.NewStyle().
		MarginTop(verticalPadding).
		MarginLeft(horizontalPadding).
		Render(helpBox)

	// Overlay on the main view
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, positioned)
}

// applyFilterMode filters the file list based on the current filter mode
func (m *Model) applyFilterMode() {
	if m.diff == nil {
		return
	}

	// Filter files based on the current mode
	filteredFiles := m.filterFiles(m.diff.Files)

	// Remember currently selected file path
	var currentPath string
	if activeFile := m.fileList.GetActiveFile(); activeFile != nil {
		currentPath = activeFile.NewPath
		if currentPath == "" {
			currentPath = activeFile.OldPath
		}
	}

	// Update file list with filtered files and set filter mode info
	m.fileList.SetFiles(filteredFiles)
	m.fileList.SetFilterMode(int(m.filterMode), len(m.diff.Files))

	// Try to restore selection to the same file
	if currentPath != "" {
		m.fileList.SelectByPath(currentPath)
	}

	// Update the diff view's filter mode
	m.diffView.SetFilterMode(tui.FilterMode(m.filterMode))

	logger.Info("applyFilterMode: mode=%s, filtered=%d/%d files",
		m.filterMode.String(), len(filteredFiles), len(m.diff.Files))
}

// filterFiles returns files that match the current filter mode
func (m *Model) filterFiles(files []*ctypes.FileDiff) []*ctypes.FileDiff {
	if m.filterMode == FilterModeNone {
		return files
	}

	var filtered []*ctypes.FileDiff
	for _, file := range files {
		gitPath := file.NewPath
		if file.IsDeleted {
			gitPath = file.OldPath
		}

		// Get conversation summary for this file
		summary, err := m.messaging.GetFileConversationSummary(gitPath)
		if err != nil {
			logger.Warn("filterFiles: error getting summary for %s: %v", gitPath, err)
			continue
		}

		switch m.filterMode {
		case FilterModeWithComments:
			// Include if file has any comments (resolved or unresolved)
			if summary.HasUnresolvedComments || summary.HasResolvedComments {
				filtered = append(filtered, file)
				logger.Debug("filterFiles: including %s (has comments)", gitPath)
			}
		case FilterModeWithUnresolved:
			// Include only if file has unresolved comments
			if summary.HasUnresolvedComments {
				filtered = append(filtered, file)
				logger.Debug("filterFiles: including %s (has unresolved)", gitPath)
			}
		}
	}

	logger.Info("filterFiles: mode=%s, files=%d->%d",
		m.filterMode.String(), len(files), len(filtered))

	return filtered
}

// getCurrentCodeVersion returns the current git commit hash as the code version
func (m *Model) getCurrentCodeVersion() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		logger.Warn("Failed to get current commit: %v", err)
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// diffStats holds statistics about a diff
type diffStats struct {
	Added   int
	Deleted int
	Moved   int
}

// computeDiffStats computes line statistics for a diff
func computeDiffStats(diff *ctypes.Diff) diffStats {
	var stats diffStats
	if diff == nil {
		return stats
	}

	// Sum up pre-computed hunk stats and track content for move detection
	addedLines := make(map[string]int)   // content -> count
	deletedLines := make(map[string]int) // content -> count

	for _, file := range diff.Files {
		for _, hunk := range file.Hunks {
			// Use pre-computed stats from hunk
			stats.Added += hunk.Stats.Added
			stats.Deleted += hunk.Stats.Deleted

			// Track line content for move detection
			for _, line := range hunk.Lines {
				switch line.Type {
				case ctypes.LineAdded:
					addedLines[line.Content]++
				case ctypes.LineDeleted:
					deletedLines[line.Content]++
				}
			}
		}
	}

	// Detect moved lines: content that appears in both added and deleted
	for content, deletedCount := range deletedLines {
		if addedCount, ok := addedLines[content]; ok {
			// Count the minimum as moved (the rest are true adds/deletes)
			moved := deletedCount
			if addedCount < moved {
				moved = addedCount
			}
			stats.Moved += moved
		}
	}

	// Adjust added/deleted to exclude moved lines
	stats.Added -= stats.Moved
	stats.Deleted -= stats.Moved

	return stats
}
