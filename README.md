# Critic

A code review tool for Git repositories with both TUI and Web interfaces.

## Features

- **TUI Interface**: Terminal-based diff viewer with syntax highlighting
- **Web Interface**: Browser-based diff viewer using htmx
- Side-by-side diff comparison
- Inline code review comments
- Real-time updates via WebSocket
- Multiple base comparison (main, origin/branch, HEAD)
- File filtering by extension
- Dark/light theme support

## Installation

```bash
go build -o critic ./cmd/critic
```

## Usage

### Terminal UI (TUI)

```bash
# Start TUI with default bases (main/master, origin/branch, HEAD)
critic tui

# Compare against specific base
critic tui main

# Compare against multiple bases
critic tui main,develop

# Only show specific directories
critic tui -- src tests

# Filter by file extension
critic tui --extensions=go,rs

# Disable animations
critic tui --no-animation
```

### Web UI

```bash
# Start web interface on default port 8080
critic webui

# Start on custom port
critic webui --port=3000

# Compare against specific base
critic webui main

# Only show specific paths
critic webui -- src tests
```

Then open http://localhost:8080 in your browser.

### Other Commands

```bash
# Start MCP server
critic mcp

# Manage conversations
critic convo

# View logs
critic log
```

## Keyboard Shortcuts

### TUI
- `Tab`: Switch between file list and diff panes
- `↑/↓` or `k/j`: Navigate up/down
- `Shift+↑/↓`: Move 10 lines
- `Space` / `Shift+Space`: Page down/up in diff
- `[` / `]`: Previous/next hunk
- `n` / `p`: Next/previous file
- `b` / `B`: Switch base
- `f` / `F`: Cycle filter mode (All / With Comments / Unresolved Only)
- `Enter`: Add comment on current line
- `?`: Show help
- `q`: Quit

### Web UI
- `?`: Toggle help overlay
- `j` / `k`: Navigate in file list
- `Tab`: Switch focus between panes
- Theme toggle button in header for dark/light mode

## Testing

### Go Tests

```bash
# Run all Go tests
go test ./...

# Run with verbose output
go test -v ./...
```

### End-to-End Tests (Puppeteer)

The e2e tests verify the web UI functionality using Puppeteer.

```bash
# First, build the critic binary
go build -o critic ./cmd/critic

# Install test dependencies
cd tests/e2e
npm install

# Run the tests
npm test
```

The e2e tests cover:
- Page load and structure
- File list rendering
- Diff display
- Theme toggle functionality
- Keyboard navigation
- API endpoints
- WebSocket connection

## Architecture

See [docs/architecture.md](docs/architecture.md) for the overall architecture.

See [docs/webui-architecture.md](docs/webui-architecture.md) for web UI specific details.

## Development

### Project Structure

```
critic/
├── cmd/critic/          # Main entry point
├── internal/
│   ├── app/             # Application logic
│   ├── cli/             # CLI command definitions
│   ├── tui/             # Terminal UI components
│   ├── webui/           # Web UI server and handlers
│   │   ├── static/      # CSS, JS assets
│   │   └── templates/   # HTML templates
│   ├── git/             # Git operations
│   ├── highlight/       # Syntax highlighting
│   └── messagedb/       # Comment storage
├── pkg/
│   ├── critic/          # Core types and interfaces
│   └── types/           # Shared types
├── tests/
│   ├── e2e/             # Puppeteer e2e tests
│   └── integration/     # Integration tests
└── docs/                # Documentation
```

### Adding New Features

1. For TUI changes, modify files in `internal/tui/`
2. For Web UI changes, modify files in `internal/webui/`
3. Update tests accordingly
4. Run `go build` and test both interfaces

## License

See LICENSE file.
