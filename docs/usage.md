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

1. Human reviewer opens `critic webui` and adds inline comments on the diff
2. AI assistant calls `get_critic_conversations(status: "unresolved")` to discover new comments
3. AI calls `get_full_critic_conversation(uuid)` to read the full conversation
4. AI addresses the feedback and calls `reply_to_critic_conversation(uuid, message)` to respond
5. Human sees AI response in the Web UI (auto-refreshes via WebSocket)
6. Conversation continues until human marks it resolved

### Tool Parameters

**get_critic_conversations**
```json
{
  "status": "unresolved"  // Optional: "unresolved", "resolved", or omit for all
}
```

**get_full_critic_conversation**
```json
{
  "uuid": "conversation-uuid-here"
}
```

**reply_to_critic_conversation**
```json
{
  "uuid": "conversation-uuid-here",
  "message": "Your response text here"
}
```
