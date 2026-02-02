# Usage

## Command Line

### Web UI

```bash
critic webui                         # Start web UI on default port
critic webui --port=8080             # Start on specific port
```

### MCP Server

```bash
critic mcp                           # Start MCP server (JSON-RPC over stdin/stdout)
```

## Web UI Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `?` | Toggle help overlay |
| `j` / `k` | Navigate in file list |
| `Tab` | Switch focus between panes |

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
