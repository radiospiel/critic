# Hacking Guide

## Directory Structure

```
critic/
├── cmd/critic/       # Entry point
├── internal/         # Core application code
│   ├── app/          # Application state machine
│   ├── cli/          # CLI parsing (Cobra)
│   ├── git/          # Git operations, file watcher, base resolver
│   ├── highlight/    # Syntax highlighting (Chroma)
│   ├── mcp/          # MCP server (JSON-RPC 2.0)
│   ├── messagedb/    # SQLite message storage
│   ├── tui/          # TUI components (Bubble Tea)
│   └── webui/        # Web UI (htmx + WebSocket)
├── pkg/              # Public packages (types, interfaces)
├── simple-go/        # Utility packages (assert, logger, must)
├── teapot/           # Custom UI framework
└── tests/            # Integration and E2E tests
```

## Architecture

### UI Framework (teapot/)

Custom widget system built on Bubble Tea with AnimationLayer for layered rendering. Widgets are composable and manage their own state.

### File Watcher

Three-stage pipeline for handling filesystem events:

```
fsnotify → eventLoop → filterLoop → debounceLoop
```

- `eventLoop`: Receives raw fsnotify events
- `filterLoop`: Filters out irrelevant events
- `debounceLoop`: Debounces rapid events into single updates

### Base Resolver

Polls git refs every 10 seconds to detect changes to branches and tags. Supports multiple base references that users can cycle through.

### Web UI

Go HTTP server serving htmx-powered pages with WebSocket connections for real-time updates. When files or comments change, the server notifies connected clients via WebSocket.

## Testing

### Test Types

| Type | Location | Purpose |
|------|----------|---------|
| Unit Tests | `*_test.go` | Test individual functions |
| Integration Tests | `tests/integration/` | Component interactions |
| E2E Tests | `tests/e2e/` | Web UI via Puppeteer |
| Profile Tests | `tests/pprof/` | Rendering performance |

### Running Tests

```bash
# All tests
go test ./...

# With race detection
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...

# E2E tests (requires npm)
cd tests/e2e && npm test
```

### Testing Conventions

Use the `assert` package instead of manual `if` checks:

```go
// Good
assert.Contains(t, conversations, conv1.ID, "expected %v in conversations", conv1.ID)

// Avoid
if !contains(conversations, conv1.ID) {
    t.Error("expected conv1 in conversations")
}
```

### Manual TUI Testing

Before completing significant TUI changes, test manually in the fixtures repo:

```bash
cd tests/integration
make fixtures
cd fixtures/repo
# Run critic and inspect rendering
```

## Code Review Process

Before completing significant code changes, request human reviewer feedback with a summary of changes. Address any feedback before proceeding.
