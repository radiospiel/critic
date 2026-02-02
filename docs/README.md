# Critic Documentation

Critic is a git diff viewer with integrated code review comments. It enables human-in-the-loop code review workflows with AI assistants through MCP (Model Context Protocol) integration.

## Documentation

- [Installation](installation.md) - How to build and install Critic
- [Usage](usage.md) - Command-line options and keyboard shortcuts
- [Design](design.md) - System architecture and communication patterns
- [Hacking](hacking.md) - Testing and development guide
- [Plans](plans.md) - Roadmap and planned features

## Quick Start

```bash
# Build and install
go install ./src/cmd/

# Start web UI
critic webui --port=8080

# Start MCP server (for AI integration)
critic mcp
```

## Features

- Web-based diff viewer with syntax highlighting
- Inline code review comments stored in SQLite
- Real-time updates via WebSocket
- MCP server for AI assistant integration
- Git and database watchers for automatic refresh
