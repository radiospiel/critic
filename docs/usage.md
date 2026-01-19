# Usage

## Command Line

### Basic Usage

```bash
critic                               # View changes against merge-base
critic main..current                 # Compare against specific base
critic merge-base,origin/main..current  # Multiple bases (cycle with 'b')
```

### Filtering

```bash
critic --extensions=go,rs            # Filter by file extension
critic main..current -- src/ tests/  # Filter by path
```

### Web UI

```bash
critic webui                         # Start web UI on default port
critic webui --port=8080             # Start on specific port
```

## TUI Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `b` | Cycle through base references |
| `Tab` | Switch focus between file list and diff view |
| `j` / `k` | Navigate down/up |
| `Up` / `Down` | Navigate up/down |
| `Space` | Page down |
| `Shift+Space` | Page up |
| `Enter` | Open comment editor on current line |
| `r` | Resolve comment |
| `?` | Show help |
| `q` | Quit |

## MCP Tools

When running as an MCP server (`critic mcp`), the following tools are available for AI assistants:

| Tool | Description |
|------|-------------|
| `get_critic_conversations` | List all conversation UUIDs in the current repository |
| `get_full_critic_conversation` | Get a full conversation with all messages by UUID |
| `reply_to_critic_conversation` | Add a reply to an existing conversation |

### Example MCP Workflow

1. Human reviewer adds inline comment in Critic TUI
2. AI assistant calls `get_critic_conversations` to discover new comments
3. AI calls `get_full_critic_conversation` to read the comment content
4. AI calls `reply_to_critic_conversation` to respond
5. Human sees AI response in Critic and can continue the conversation
