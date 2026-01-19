# Critic Documentation

Critic is a terminal-based git diff viewer with syntax highlighting, interactive navigation, and integrated code review comments. It enables human-in-the-loop code review workflows with AI assistants through MCP (Model Context Protocol) integration.

## Documentation

- [Installation](installation.md) - How to build and install Critic
- [Usage](usage.md) - Command-line options and keyboard shortcuts
- [Hacking](hacking.md) - Architecture, testing, and development guide
- [Plans](plans.md) - Roadmap and planned features

## Quick Start

```bash
# Build and install
go install ./cmd/critic

# View changes against merge-base
critic

# Compare against a specific base
critic main..current

# Start web UI
critic webui --port=8080
```

## Features

- Syntax-highlighted diff viewing in the terminal
- Interactive file navigation with keyboard shortcuts
- Inline code review comments stored in SQLite
- Multiple base reference support (cycle with 'b')
- File extension and path filtering
- Web UI with real-time updates via WebSocket
- MCP server for AI assistant integration
