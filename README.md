# Critic

A code review tool for Git repositories with Web and MCP interfaces, enabling human-in-the-loop code review workflows with AI assistants.

## Features

- **Web Interface**: Browser-based diff viewer (React)
- **MCP Server**: AI assistant integration via Model Context Protocol
- Side-by-side diff comparison
- Inline code review comments with threading
- Real-time updates via WebSocket
- Multiple base comparison (main, origin/branch, HEAD)
- File filtering by extension
- Dark/light theme support

## Installation

```bash
git clone https://github.com/radiospiel/critic.git
cd critic
make build
```

To install system-wide: `make install`

## Usage

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

### MCP Server Integration

To enable AI-assisted code review with Claude Code:

```bash
# Add critic as an MCP server
claude mcp add critic -- /path/to/critic mcp
```

This gives AI assistants access to:
- `get_critic_conversations` - List pending review conversations
- `get_full_critic_conversation` - Read conversation details
- `reply_to_critic_conversation` - Respond to reviewer feedback

See [docs/installation.md](docs/installation.md) for detailed setup and HITL workflow configuration.

## Keyboard Shortcuts (Web UI)

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
# Build critic first
make build

# Run e2e tests
cd tests/e2e
npm install
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

See [docs/design.md](docs/design.md) for the system architecture and communication patterns between WebUI, MCP server, and other components.

## Development

### Project Structure

```
critic/
├── src/
│   ├── cmd/             # Main entry point
│   ├── cli/             # CLI command definitions (Cobra)
│   ├── api/             # API server (Connect/gRPC)
│   ├── webui/           # Web UI frontend (React)
│   ├── git/             # Git operations, file watcher
│   ├── mcp/             # MCP server (JSON-RPC 2.0)
│   ├── messagedb/       # SQLite message storage
│   ├── config/          # Configuration
│   └── pkg/             # Core types and interfaces
├── simple-go/           # Utility packages (assert, logger)
├── tests/
│   ├── e2e/             # Puppeteer e2e tests
│   └── integration/     # Integration tests
├── agents/              # AI agent configuration
└── docs/                # Documentation
```

### Adding New Features

1. For Web UI changes, modify files in `src/webui/`
2. For API changes, modify files in `src/api/`
3. For MCP server changes, modify files in `src/mcp/`
4. Update tests accordingly
5. Run `go build` and test the interface

## License

See LICENSE file.
