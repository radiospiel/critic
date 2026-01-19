package teapot

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// AppDelegate allows applications to customize App behavior.
// Implement this interface to handle app-specific keys and messages.
type AppDelegate interface {
	// Init is called once at startup. Return commands to run.
	Init() tea.Cmd

	// HandleKey processes key events not handled by the framework.
	// Called after focus manager handles tab/shift-tab and modal routing.
	// Return true if the key was handled.
	HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd)

	// HandleMessage processes non-key messages (e.g., custom message types).
	// Return a command to execute, or nil.
	HandleMessage(msg tea.Msg) tea.Cmd

	// ShouldQuit returns true if the given key should trigger quit.
	// Default keys (q, ctrl+c) are checked by App; this allows additional quit keys.
	ShouldQuit(msg tea.KeyMsg) bool
}

// ViewDecorator is an optional interface that delegates can implement
// to add overlays (modals, help screens) on top of the compositor output.
type ViewDecorator interface {
	// View receives the base compositor view and returns the final view.
	// Use this to overlay modals, help screens, error messages, etc.
	View(baseView string) string
}

// App is a general-purpose TUI application framework built on bubbletea.
// It provides:
// - Terminal setup (alternate screen, line wrap control)
// - Compositor-based rendering
// - Focus management with tab/shift-tab navigation
// - Modal dialog support
// - Tick loop for animations
type App struct {
	compositor   *Compositor
	focusManager *FocusManager
	delegate     AppDelegate

	width, height int
	ready         bool
}

// NewApp creates a new App with the given root view and delegate.
// The delegate handles app-specific behavior; pass nil for defaults.
func NewApp(root View, delegate AppDelegate) *App {
	compositor := NewCompositor(root)
	focusManager := NewFocusManager(root)

	return &App{
		compositor:   compositor,
		focusManager: focusManager,
		delegate:     delegate,
	}
}

// Compositor returns the app's compositor for direct access if needed.
func (a *App) Compositor() *Compositor {
	return a.compositor
}

// FocusManager returns the app's focus manager.
func (a *App) FocusManager() *FocusManager {
	return a.focusManager
}

// Size returns the current terminal size.
func (a *App) Size() (width, height int) {
	return a.width, a.height
}

// Init implements tea.Model. Sets up terminal and starts tick loop.
func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{
		disableTerminalLineWrap,
		// Start compositor tick loop
		tea.Tick(ComposerTickInterval, func(_ time.Time) tea.Msg {
			return ComposerTickMsg{}
		}),
	}

	// Call delegate init if provided
	if a.delegate != nil {
		if cmd := a.delegate.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

// Update implements tea.Model. Handles all message routing.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Check for quit keys first
		if a.shouldQuit(msg) {
			return a, tea.Sequence(enableTerminalLineWrap, tea.Quit)
		}

		// If a modal is active, route all keys through focus manager
		if a.focusManager.HasModal() {
			_, cmd := a.focusManager.HandleKey(msg)
			cmds = append(cmds, cmd)
			return a, tea.Batch(cmds...)
		}

		// Delegate gets first chance to handle keys (including tab/shift-tab)
		if a.delegate != nil {
			if handled, cmd := a.delegate.HandleKey(msg); handled {
				cmds = append(cmds, cmd)
				return a, tea.Batch(cmds...)
			}
		}

		// Default tab/shift-tab handling if delegate didn't handle it
		switch msg.String() {
		case "tab":
			a.focusManager.FocusNext()
			return a, nil
		case "shift+tab":
			a.focusManager.FocusPrev()
			return a, nil
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.compositor.Resize(msg.Width, msg.Height)

		if !a.ready {
			a.ready = true
		}

		// Clear screen on resize to avoid artifacts
		return a, tea.ClearScreen

	case ComposerTickMsg:
		// Notify all tick subscribers
		NotifyGlobalTickSubscribers()

		// Continue ticking
		cmds = append(cmds, tea.Tick(ComposerTickInterval, func(_ time.Time) tea.Msg {
			return ComposerTickMsg{}
		}))

		return a, tea.Batch(cmds...)
	}

	// Route other messages to delegate
	if a.delegate != nil {
		if cmd := a.delegate.HandleMessage(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

// View implements tea.Model. Renders using the compositor.
func (a *App) View() string {
	if !a.ready {
		return "Initializing..."
	}

	view := a.compositor.Render()

	// If delegate implements ViewDecorator, let it add overlays
	if decorator, ok := a.delegate.(ViewDecorator); ok {
		view = decorator.View(view)
	}

	return view
}

// shouldQuit checks if the key should trigger application quit.
func (a *App) shouldQuit(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "q", "ctrl+c":
		return true
	}

	// Check delegate for additional quit keys
	if a.delegate != nil && a.delegate.ShouldQuit(msg) {
		return true
	}

	return false
}

// disableTerminalLineWrap sends escape sequences to set up the terminal.
func disableTerminalLineWrap() tea.Msg {
	fmt.Print("\x1b[?7l")    // DECAWM - disable auto wrap mode
	fmt.Print("\x1b[?1049h") // Use alternate screen buffer
	return nil
}

// enableTerminalLineWrap sends escape sequences to restore the terminal.
func enableTerminalLineWrap() tea.Msg {
	fmt.Print("\x1b[?1049l") // Exit alternate screen buffer
	fmt.Print("\x1b[?7h")    // DECAWM - enable auto wrap mode
	return nil
}

// Run is a convenience function to run an App as a bubbletea program.
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunWithOptions runs the App with custom bubbletea options.
func (a *App) RunWithOptions(opts ...tea.ProgramOption) error {
	p := tea.NewProgram(a, opts...)
	_, err := p.Run()
	return err
}
