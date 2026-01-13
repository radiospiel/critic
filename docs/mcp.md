# MCP Server

Model Context Protocol server for Claude Code integration. This enables Claude to interact with code review comments stored in Critic's database.

## Architecture

```
┌─────────────────┐     stdio/JSON-RPC    ┌─────────────────┐
│  Claude Code    │◄────────────────────►│  critic mcp     │
└─────────────────┘                       └────────┬────────┘
                                                   │
                                          ┌────────▼────────┐
                                          │   .critic.db    │
                                          │   (SQLite)      │
                                          └─────────────────┘
```

The MCP server communicates with Claude Code via stdio using JSON-RPC 2.0 protocol. It reads and writes conversations from the SQLite database at the git repository root.

## Configuration

Add to your Claude Code MCP settings:

**Global** (`~/.claude/settings.json`):
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

**Project** (`.claude/settings.json`):
```json
{
  "mcpServers": {
    "critic": {
      "command": "/path/to/critic",
      "args": ["mcp"]
    }
  }
}
```

## Tools

### get_critic_conversations

Returns a list of conversation UUIDs. Use this to discover reviewer feedback.

**Input:**
```json
{
  "status": "unresolved"  // Optional: "unresolved" or "resolved"
}
```

**Output:** JSON array of UUIDs
```json
["550e8400-e29b-41d4-a716-446655440000", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"]
```

### get_full_critic_conversation

Returns the complete conversation with all messages.

**Input:**
```json
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000"  // Required
}
```

**Output:** Formatted conversation
```
Conversation: 550e8400-e29b-41d4-a716-446655440000
Status: new
Location: main.go:42
Code Version: abc123def

[human] 2025-01-13 10:30:00
This function needs error handling for the nil case.

[ai] 2025-01-13 10:35:00
I've added a nil check at line 45.
```

### reply_to_critic_conversation

Adds a reply to an existing conversation.

**Input:**
```json
{
  "uuid": "550e8400-e29b-41d4-a716-446655440000",  // Required
  "message": "I've addressed this by adding error handling."  // Required
}
```

**Output:** Confirmation with reply UUID

## Workflow

1. Human adds comment in Critic TUI (stored in `.critic.db`)
2. Claude calls `get_critic_conversations` with `status: "unresolved"`
3. Claude calls `get_full_critic_conversation` for each UUID
4. Claude addresses feedback and calls `reply_to_critic_conversation`
5. Human reviews reply in Critic TUI
6. Human resolves conversation when satisfied

## Prompting Claude

Add to your `CLAUDE.md`:

```markdown
Before completing any significant code changes, call get_review_feedback with
a summary of what you've done. Wait for reviewer approval before proceeding.
Address any feedback in subsequent iterations.
```

## Protocol Details

The server implements MCP protocol version 2024-11-05 over stdio using JSON-RPC 2.0.

**Supported methods:**

| Method | Description |
|--------|-------------|
| `initialize` | Initialize the server |
| `initialized` | Notification that initialization is complete |
| `tools/list` | List available tools |
| `tools/call` | Call a tool |
| `ping` | Health check |

## Related Documentation

- [Summary](summary.md) - Project overview
- [Architecture](architecture.md) - Database schema and interfaces
- [Installation](installation.md) - Setup and configuration
