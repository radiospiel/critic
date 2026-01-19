# Critic

Critic is a terminal-based git diff viewer with syntax highlighting, interactive navigation, and integrated code review comments. It enables human-in-the-loop code review workflows with AI assistants through MCP (Model Context Protocol) integration.

## Features

- **Interactive diff viewing** - Navigate files and hunks with keyboard shortcuts
- **Multiple base comparison** - Compare against merge-base, branches, or commits; cycle between bases with `b`
- **Syntax highlighting** - Language-aware highlighting for 60+ file types
- **Live file watching** - Automatic refresh when files change on disk
- **Code review comments** - Add comments to specific lines, stored in SQLite database
- **Threaded conversations** - Comments support replies and resolution status
- **MCP integration** - Expose comments to AI assistants for human-in-the-loop review

## Quick Start

```bash
# View changes in working directory against merge-base
critic

# Compare against specific base
critic main..current

# Compare against multiple bases (cycle with 'b')
critic merge-base,origin/main,HEAD..current

# Filter by file extension
critic --extensions=go,rs

# Filter by path
critic main..current -- src/ tests/
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `b` | Cycle through base references |
| `Tab` | Switch focus between file list and diff view |
| `j/k` or `Up/Down` | Navigate up/down |
| `Space` | Page down |
| `Shift+Space` | Page up |
| `Enter` | Open comment editor on current line |
| `r` | Resolve comment at cursor |
| `?` | Show help |
| `q` or `Ctrl+C` | Quit |

## Architecture Overview

```
Terminal UI (Bubble Tea + Teapot)
├── FileListWidget (left pane)
├── DiffViewModel (right pane)
└── CommentEditor (modal)

Application Layer
├── Git Watcher (file change detection)
├── Base Resolver (ref polling)
└── Diff Engine (parsing)

Data Layer
├── Message Database (SQLite)
└── Git Commands

Integration
└── MCP Server (Claude AI)
```

## MCP Integration

Critic provides an MCP server that exposes code review comments to AI assistants:

```json
{
  "mcpServers": {
    "critic": {
      "command": "critic",
      "args": ["mcp"]
    }
  }
}
```

**Available tools:**
- `get_critic_conversations` - List conversation UUIDs
- `get_full_critic_conversation` - Get full conversation with messages
- `reply_to_critic_conversation` - AI adds reply to conversation

## Related Documentation

- [Architecture](architecture.md) - Code layout, data structures, interfaces
- [Installation](installation.md) - Build and configuration
