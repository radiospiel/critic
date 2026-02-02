# Critic Documentation

Critic is a mcp and web server which offers a git diff viewer and integrates code review comments. It enables human-in-the-loop code review workflows with AI assistants through MCP (Model Context Protocol) integration. It allows humans to review code changes made by the agent, and to ask for adjustments, without going through a github. This allows for a faster turn-around between changes and review, allowing the agent to actively employ a human-in-the-loop review process.

**Note that critic is designed as a single-user experience.** A typical scenario runs the critic mcp and http servers on the same machine that also runs the coding agent, and only listens on localhost. It is important to not deploy critic in an unsecured environment, since the web client has access to the entire tree of source files.

## Documentation

- [Installation](installation.md) - How to build and install Critic
- [Usage](usage.md) - Command-line options and keyboard shortcuts
- [Design](design.md) - System architecture and communication patterns
- [Hacking](hacking.md) - Testing and development guide
- [Plans](plans.md) - Roadmap and planned features

## Quick Start

```bash
# Clone and build
git clone https://github.com/radiospiel/critic.git
cd critic
make build

# Start web UI
./critic webui --port=8080

# Start MCP server (for AI integration)
./critic mcp
```

## Features

- Web-based diff viewer with syntax highlighting
- Inline code review comments stored in SQLite
- Real-time updates via WebSocket
- MCP server for AI assistant integration
- Git and database watchers for automatic refresh
