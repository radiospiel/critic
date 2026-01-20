package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"git.15b.it/eno/critic/internal/config"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/matrix"
	"git.15b.it/eno/critic/internal/messagedb"
	"git.15b.it/eno/critic/internal/tui"
	"git.15b.it/eno/critic/internal/version"
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

	// Create delegate (critic-specific logic)
	delegate := NewDelegate(args)

	// Create and run the application using teapot.App
	app := teapot.NewApp(delegate.mainLayout, delegate)
	delegate.app = app // Give delegate access to app for focus manager

	return app.Run()
}

// Delegate implements teapot.AppDelegate for critic-specific behavior
type Delegate struct {
	app           *teapot.App // Set after app creation
	fileList      *tui.FileListView
	diffView      *tui.DiffView
	commentEditor tui.CommentEditor
	statusBar     *tui.StatusBarView
	mainLayout    *tui.MainView
	layout        tui.LayoutView // Legacy layout for pane focus tracking
	diff          *ctypes.Diff
	bases         []string          // List of base refs
	currentBase   int               // Index of current base
	paths         []string          // Paths to diff
	extensions    []string          // File extensions to include
	resolver      *git.BaseResolver // Base resolver with polling
	messaging     critic.Messaging  // Messaging interface for conversations
	filterMode    FilterMode        // Current filter mode (None, WithComments, WithUnresolved)
	noAnimation bool              // Whether animations are disabled
	err         error
	showHelp    bool                 // Whether to show help screen
	screensaver *matrix.Screensaver  // Matrix screensaver
	gitRoot     string               // Git repository root path
}

// NewDelegate creates a new critic delegate
func NewDelegate(args *Args) *Delegate {
	logger.Info("NewDelegate: Creating delegate with %d paths, %d bases", len(args.Paths), len(args.Bases))
	diffView := tui.NewDiffView()

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

	// Create the Matrix screensaver
	screensaver := matrix.NewScreensaver()

	// Check if this is the first run for this version
	showScreensaver := version.IsFirstRunForVersion(gitRoot)
	if showScreensaver {
		logger.Info("First run for version %s, will show screensaver", version.Version())
	}

	d := &Delegate{
		fileList:      fileList,
		diffView:      diffView,
		commentEditor: tui.NewCommentEditor(),
		statusBar:     statusBar,
		mainLayout:    mainLayout,
		layout:        tui.NewLayoutView(),
		bases:         args.Bases,
		currentBase:   0, // Start with first base
		paths:         args.Paths,
		extensions:    args.Extensions,
		messaging:   mdb,
		noAnimation: args.NoAnimation,
		screensaver: screensaver,
		gitRoot:     gitRoot,
	}

	// Set up screensaver done callback to mark version as seen
	screensaver.SetOnDone(func() {
		if err := version.MarkVersionSeen(gitRoot); err != nil {
			logger.Warn("Failed to mark version as seen: %v", err)
		}
	})

	return d
}

// Init implements teapot.AppDelegate
func (d *Delegate) Init() tea.Cmd {
	logger.Info("Init: Starting application initialization")

	// Enable compositor debug logging
	teapot.CompositorDebug = true

	// Check terminal color support
	checkTerminalColors()

	// Check if we should show the screensaver on startup
	if version.IsFirstRunForVersion(d.gitRoot) {
		// We'll start the screensaver after we get the window size
		d.screensaver.SetOnDone(func() {
			if err := version.MarkVersionSeen(d.gitRoot); err != nil {
				logger.Warn("Failed to mark version as seen: %v", err)
			}
		})
	}

	return tea.Batch(
		initBaseResolverCmd(d),
		loadDiffCmd(d),
	)
}

// HandleKey implements teapot.AppDelegate - handles critic-specific keys
func (d *Delegate) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	var cmds []tea.Cmd

	// If screensaver is active, route keys to it (any key dismisses it)
	if d.screensaver.IsActive() {
		return d.screensaver.HandleKey(msg)
	}

	switch msg.String() {
	case "tab", "shift+tab":
		// Override teapot.App's focus handling to use legacy pane-based focus
		d.layout.ToggleFocus()
		d.fileList.SetFocused(d.layout.GetFocusedPane() == tui.FileListPane)
		d.diffView.SetFocused(d.layout.GetFocusedPane() == tui.DiffViewPane)
		return true, nil

	case "b":
		// Cycle through bases
		d.currentBase = (d.currentBase + 1) % len(d.bases)
		logger.Info("Update: Switching to base %d: %s", d.currentBase, d.bases[d.currentBase])
		return true, loadDiffCmd(d)

	case "f", "F":
		// Cycle through filter modes (works from both file list and diff pane)
		d.filterMode = (d.filterMode + 1) % 3
		logger.Info("Update: Switching to filter mode %d: %s", d.filterMode, d.filterMode.String())
		// Update status bar
		d.statusBar.SetFilter(d.filterMode.String())
		// Re-apply filter to file list and update diff view
		d.applyFilterMode()
		cmd := d.diffView.SetFile(d.fileList.GetActiveFile())
		cmds = append(cmds, cmd)
		d.mainLayout.Repaint()
		return true, tea.Batch(cmds...)

	case " ": // Space - page down diff view regardless of focus
		cmd := d.diffView.ScrollPageDown()
		cmds = append(cmds, cmd)
		d.mainLayout.Repaint()
		return true, tea.Batch(cmds...)

	case "shift+ ": // Shift+Space - page up diff view regardless of focus
		cmd := d.diffView.ScrollPageUp()
		cmds = append(cmds, cmd)
		d.mainLayout.Repaint()
		return true, tea.Batch(cmds...)

	case "?":
		// Toggle help screen
		d.showHelp = !d.showHelp
		return true, nil

	case "m":
		// Toggle Matrix screensaver
		if d.screensaver.IsActive() {
			d.screensaver.Stop()
		} else {
			width, height := d.app.Size()
			d.screensaver.Start(width, height)
		}
		return true, nil

	case "enter":
		// Activate comment editor when focused on diff view
		if d.layout.GetFocusedPane() == tui.DiffViewPane {
			activeFile := d.fileList.GetActiveFile()
			if activeFile != nil {
				cursorLine := d.diffView.GetCursorLine()
				isCommentLine, sourceLine := d.diffView.IsCommentLine(cursorLine)

				var lineNum int
				var conv *critic.Conversation

				if isCommentLine {
					lineNum = sourceLine
					uuid := d.diffView.GetConversationUUIDAtLine(cursorLine)
					if uuid != "" {
						if c, err := d.messaging.GetFullConversation(uuid); err == nil {
							conv = c
						}
					}
				} else {
					lineNum = d.diffView.GetSourceLine(cursorLine)
					if lineNum == 0 {
						return true, nil
					}
					conv = nil
				}

				cmd := d.commentEditor.ActivateWithConversation(lineNum, conv)
				// Set the focus manager on the dialog and register as modal
				fm := d.app.FocusManager()
				d.commentEditor.SetFocusManager(fm)
				fm.SetModal(&d.commentEditor)
				cmds = append(cmds, cmd)
				return true, tea.Batch(cmds...)
			}
		}
		return false, nil

	default:
		// Route key messages to focused pane
		if d.layout.GetFocusedPane() == tui.FileListPane {
			prevFile := d.fileList.GetActiveFile()
			_, cmd := d.fileList.HandleKey(msg)
			if d.fileList.GetActiveFile() != prevFile {
				newFile := d.fileList.GetActiveFile()
				if newFile != nil {
					logger.Info("File selected: %s", newFile.NewPath)
				}
				setFileCmd := d.diffView.SetFile(newFile)
				cmds = append(cmds, cmd, setFileCmd)
				d.mainLayout.Repaint()
			} else {
				cmds = append(cmds, cmd)
				d.mainLayout.Repaint()
			}
			return true, tea.Batch(cmds...)
		} else {
			cmd := d.diffView.Update(msg)
			cmds = append(cmds, cmd)
			d.mainLayout.Repaint()
			return true, tea.Batch(cmds...)
		}
	}
}

// HandleMessage implements teapot.AppDelegate - handles critic-specific messages
func (d *Delegate) HandleMessage(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Handle window resize for legacy layout and comment editor
		d.layout.SetSize(msg.Width, msg.Height)

		// Update comment editor size (80% width, centered)
		editorWidth := msg.Width * 80 / 100
		editorHeight := msg.Height - 6
		d.commentEditor.SetSize(editorWidth, editorHeight)

		// Start screensaver on first window size message if this is first run
		if version.IsFirstRunForVersion(d.gitRoot) && !d.screensaver.IsActive() {
			logger.Info("Starting screensaver with size %dx%d", msg.Width, msg.Height)
			d.screensaver.Start(msg.Width, msg.Height)
		}

	case diffLoadedMsg:
		logger.Info("Update: Received diffLoadedMsg")
		d.diff = msg.diff
		d.err = msg.err
		if d.diff != nil {
			logger.Info("Update: Diff loaded with %d files", len(d.diff.Files))
			d.applyFilterMode()

			if len(d.bases) > 0 {
				d.statusBar.SetBase(d.bases[d.currentBase])
			}
			d.statusBar.SetFilter(d.filterMode.String())
			stats := computeDiffStats(d.diff)
			d.statusBar.SetDiffStats(stats.Added, stats.Deleted, stats.Moved)

			cmd := d.diffView.SetFile(d.fileList.GetActiveFile())
			cmds = append(cmds, cmd)
			d.mainLayout.Repaint()
		} else if d.err != nil {
			logger.Error("Update: Diff loading failed: %v", d.err)
		}

	case baseResolverInitializedMsg:
		logger.Info("Update: BaseResolver initialized")
		d.resolver = msg.resolver

	case baseChangedMsg:
		logger.Info("Update: Received baseChangedMsg, reloading diff")
		return loadDiffCmd(d)

	case tui.RequestNextFileMsg:
		if d.fileList.SelectNext() {
			cmd := d.diffView.SetFile(d.fileList.GetActiveFile())
			cmds = append(cmds, cmd)
		}

	case tui.RequestPrevFileMsg:
		if d.fileList.SelectPrev() {
			d.diffView.SetGotoBottomOnLoad()
			cmd := d.diffView.SetFile(d.fileList.GetActiveFile())
			cmds = append(cmds, cmd)
		}

	case tui.CommentSavedMsg:
		activeFile := d.fileList.GetActiveFile()
		if activeFile != nil && msg.Comment != "" {
			filePath := activeFile.NewPath
			if filePath == "" {
				filePath = activeFile.OldPath
			}

			codeVersion := d.getCurrentCodeVersion()
			cursorLine := d.diffView.GetCursorLine()
			existingUUID := d.diffView.GetConversationUUIDAtLine(cursorLine)

			if existingUUID != "" {
				replyMsg, err := d.messaging.ReplyToConversation(
					existingUUID,
					msg.Comment,
					critic.AuthorHuman,
				)
				if err != nil {
					logger.Error("Failed to create reply: %v", err)
					return nil
				}
				logger.Info("Created reply %s to conversation %s", replyMsg.UUID, existingUUID)
			} else {
				context := git.GetLineContext(filePath, msg.LineNum, codeVersion)
				conversation, err := d.messaging.CreateConversation(
					critic.AuthorHuman,
					msg.Comment,
					filePath,
					msg.LineNum,
					codeVersion,
					context,
				)
				if err != nil {
					logger.Error("Failed to create conversation: %v", err)
					return nil
				}
				logger.Info("Created conversation %s at %s:%d", conversation.UUID, filePath, msg.LineNum)
			}

			cmd := d.diffView.RefreshFile()
			cmds = append(cmds, cmd)
		}

	default:
		// Route other messages to diff view and comment editor
		cmd := d.diffView.Update(msg)
		cmds = append(cmds, cmd)

		editorCmd := d.commentEditor.Update(msg)
		cmds = append(cmds, editorCmd)

		d.mainLayout.Repaint()
	}

	return tea.Batch(cmds...)
}

// ShouldQuit implements teapot.AppDelegate
func (d *Delegate) ShouldQuit(msg tea.KeyMsg) bool {
	return false // Use default quit keys (q, ctrl+c)
}

// View returns the rendered view - called by teapot.App to render overlays
func (d *Delegate) View(baseView string) string {
	width, height := d.app.Size()

	if d.err != nil {
		return renderError(d.err)
	}

	// If screensaver is active, render it instead of the base view
	if d.screensaver.IsActive() {
		return d.renderScreensaver(width, height)
	}

	view := baseView

	// Debug: log compositor output
	logger.Info("View: size=%dx%d, view len=%d, lines=%d", width, height, len(view), len(strings.Split(view, "\n")))

	// Overlay comment editor if active
	if d.commentEditor.IsActive() {
		editorBuf := teapot.NewBuffer(d.commentEditor.Width(), d.commentEditor.Height())
		editorSub := teapot.NewSubBuffer(editorBuf, editorBuf.Bounds())
		d.commentEditor.Render(editorSub)
		editorView := editorBuf.RenderToString()
		editorLines := strings.Split(editorView, "\n")

		leftPadding := width * 10 / 100
		lines := strings.Split(view, "\n")
		startLine := (len(lines) - len(editorLines)) / 2
		if startLine < 0 {
			startLine = 0
		}

		for i, editorLine := range editorLines {
			lineIdx := startLine + i
			if lineIdx < len(lines) {
				paddedLine := strings.Repeat(" ", leftPadding) + editorLine
				if len([]rune(paddedLine)) < width {
					paddedLine += strings.Repeat(" ", width-len([]rune(paddedLine)))
				}
				lines[lineIdx] = paddedLine
			}
		}
		return strings.Join(lines, "\n")
	}

	// Overlay help screen if showing
	if d.showHelp {
		return d.renderHelpOverlay(view, width, height)
	}

	return view
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
	term := os.Getenv("TERM")
	if term == "" {
		fmt.Fprintf(os.Stderr, "Warning: TERM environment variable not set\n")
		fmt.Fprintf(os.Stderr, "Syntax highlighting may not work correctly\n")
		return
	}

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

func initBaseResolverCmd(d *Delegate) tea.Cmd {
	return func() tea.Msg {
		resolver, err := git.NewBaseResolver(d.bases, "HEAD", func() {
			logger.Info("BaseResolver: Bases changed, triggering reload")
		})
		if err != nil {
			logger.Error("Failed to initialize base resolver: %v", err)
			return nil
		}
		return baseResolverInitializedMsg{resolver: resolver}
	}
}

func loadDiffCmd(d *Delegate) tea.Cmd {
	return func() tea.Msg {
		baseName := d.bases[d.currentBase]

		var baseCommit string
		if d.resolver != nil {
			sha, ok := d.resolver.GetResolvedBase(baseName)
			if !ok {
				resolvedSHA, err := resolveBase(baseName)
				if err != nil {
					return diffLoadedMsg{diff: nil, err: fmt.Errorf("failed to resolve base %s: %w", baseName, err)}
				}
				baseCommit = resolvedSHA
			} else {
				baseCommit = sha
			}
		} else {
			resolvedSHA, err := resolveBase(baseName)
			if err != nil {
				return diffLoadedMsg{diff: nil, err: fmt.Errorf("failed to resolve base %s: %w", baseName, err)}
			}
			baseCommit = resolvedSHA
		}

		targetCommit, err := git.ResolveRef("HEAD")
		if err != nil {
			return diffLoadedMsg{diff: nil, err: fmt.Errorf("failed to resolve HEAD: %w", err)}
		}

		logger.Info("loadDiffCmd: Loading diff from %s (%s) to HEAD (%s)", baseName, baseCommit, targetCommit)

		diff, err := git.GetDiffBetween(baseCommit, targetCommit, d.paths)
		return diffLoadedMsg{diff: diff, err: err}
	}
}

func resolveBase(base string) (string, error) {
	if git.IsCommitSHA(base) {
		return git.ResolveRef(base)
	}

	baseSHA, err := git.ResolveRef(base)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref %s: %w", base, err)
	}

	mergeBase, err := git.GetMergeBaseBetween("HEAD", baseSHA)
	if err != nil {
		return "", fmt.Errorf("failed to get merge base with %s: %w", base, err)
	}

	return mergeBase, nil
}

// renderError renders an error message
func renderError(err error) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true).
		Padding(1, 2).
		Render(fmt.Sprintf("Error: %s", err))
}

// renderScreensaver renders the Matrix screensaver
func (d *Delegate) renderScreensaver(width, height int) string {
	buf := teapot.NewBuffer(width, height)
	sub := teapot.NewSubBuffer(buf, buf.Bounds())
	d.screensaver.Render(sub)
	return buf.RenderToString()
}

// renderHelpOverlay renders the help screen overlay
func (d *Delegate) renderHelpOverlay(underlay string, width, height int) string {
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
    m             Matrix screensaver
    ?             Toggle this help screen
    q             Quit

  Press ? to close this help
`

	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(60).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252"))

	helpBox := helpStyle.Render(helpContent)

	verticalPadding := (height - lipgloss.Height(helpBox)) / 2
	horizontalPadding := (width - lipgloss.Width(helpBox)) / 2

	if verticalPadding < 0 {
		verticalPadding = 0
	}
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	positioned := lipgloss.NewStyle().
		MarginTop(verticalPadding).
		MarginLeft(horizontalPadding).
		Render(helpBox)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, positioned)
}

// applyFilterMode filters the file list based on the current filter mode
func (d *Delegate) applyFilterMode() {
	if d.diff == nil {
		return
	}

	filteredFiles := d.filterFiles(d.diff.Files)

	var currentPath string
	if activeFile := d.fileList.GetActiveFile(); activeFile != nil {
		currentPath = activeFile.NewPath
		if currentPath == "" {
			currentPath = activeFile.OldPath
		}
	}

	d.fileList.SetFiles(filteredFiles)
	d.fileList.SetFilterMode(int(d.filterMode), len(d.diff.Files))

	if currentPath != "" {
		d.fileList.SelectByPath(currentPath)
	}

	d.diffView.SetFilterMode(tui.FilterMode(d.filterMode))

	logger.Info("applyFilterMode: mode=%s, filtered=%d/%d files",
		d.filterMode.String(), len(filteredFiles), len(d.diff.Files))
}

// filterFiles returns files that match the current filter mode
func (d *Delegate) filterFiles(files []*ctypes.FileDiff) []*ctypes.FileDiff {
	if d.filterMode == FilterModeNone {
		return files
	}

	var filtered []*ctypes.FileDiff
	for _, file := range files {
		gitPath := file.NewPath
		if file.IsDeleted {
			gitPath = file.OldPath
		}

		summary, err := d.messaging.GetFileConversationSummary(gitPath)
		if err != nil {
			logger.Warn("filterFiles: error getting summary for %s: %v", gitPath, err)
			continue
		}

		switch d.filterMode {
		case FilterModeWithComments:
			if summary.HasUnresolvedComments || summary.HasResolvedComments {
				filtered = append(filtered, file)
				logger.Debug("filterFiles: including %s (has comments)", gitPath)
			}
		case FilterModeWithUnresolved:
			if summary.HasUnresolvedComments {
				filtered = append(filtered, file)
				logger.Debug("filterFiles: including %s (has unresolved)", gitPath)
			}
		}
	}

	logger.Info("filterFiles: mode=%s, files=%d->%d",
		d.filterMode.String(), len(files), len(filtered))

	return filtered
}

// getCurrentCodeVersion returns the current git commit hash as the code version
func (d *Delegate) getCurrentCodeVersion() string {
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

	addedLines := make(map[string]int)
	deletedLines := make(map[string]int)

	for _, file := range diff.Files {
		for _, hunk := range file.Hunks {
			stats.Added += hunk.Stats.Added
			stats.Deleted += hunk.Stats.Deleted

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

	for content, deletedCount := range deletedLines {
		if addedCount, ok := addedLines[content]; ok {
			moved := deletedCount
			if addedCount < moved {
				moved = addedCount
			}
			stats.Moved += moved
		}
	}

	stats.Added -= stats.Moved
	stats.Deleted -= stats.Moved

	return stats
}
