# Hacking Guide

## Directory Structure

```
critic/
├── src/
│   ├── cmd/          # Entry point
│   ├── cli/          # CLI parsing (Cobra)
│   ├── api/          # API server (Connect/gRPC)
│   ├── git/          # Git operations, file watcher
│   ├── mcp/          # MCP server (JSON-RPC 2.0)
│   ├── messagedb/    # SQLite message storage, db watcher
│   ├── webui/        # Web UI frontend (React)
│   ├── config/       # Configuration
│   └── pkg/          # Core types and interfaces
├── simple-go/        # Utility packages (assert, logger, must)
├── agents/           # AI agent configuration and logs
└── tests/            # Integration and E2E tests
```

## Architecture

See [design.md](design.md) for a detailed description of how components communicate.

### Git Watcher (`src/git/git_watcher.go`)

Uses fsnotify to monitor the `.git/` directory for changes. Implements debouncing (100ms default) to prevent notification spam from rapid file changes.

### DB Watcher (`src/messagedb/db_watcher.go`)

Polls the `_db_mtime` table to detect database changes made by external processes (like the MCP server). Uses fresh connections to ensure cross-process visibility.

### API Server (`src/api/server/`)

Connect/gRPC server providing:
- HTTP API for the web frontend
- WebSocket hub for real-time updates
- Integration with git and database watchers

### Web UI (`src/webui/`)

React frontend that connects to the API server. Receives real-time updates via WebSocket when files or comments change.

## Testing

### Test Types

| Type | Location | Purpose |
|------|----------|---------|
| Unit Tests | `*_test.go` | Test individual functions |
| Integration Tests | `tests/integration/` | Component interactions |
| E2E Tests | `tests/e2e/` | Web UI via Puppeteer |

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

## Code Review Process

Before completing significant code changes, request human reviewer feedback with a summary of changes. Address any feedback before proceeding.
