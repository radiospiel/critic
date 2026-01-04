# HITL MCP Server

Human-in-the-Loop MCP server for Claude Code integration. This enables Claude Code to request feedback from a human reviewer during code changes.

## Architecture

```
┌─────────────────┐     stdio/JSON-RPC    ┌─────────────────┐
│  Claude Code    │◄────────────────────►│  critic mcp     │
└─────────────────┘                       └────────┬────────┘
                                                   │ Unix socket
                                          ┌────────▼────────┐
                                          │ critic review   │
                                          │ (or nc -U ...)  │
                                          └─────────────────┘
```

The MCP server communicates with Claude Code via stdio using JSON-RPC 2.0 protocol, and with the human reviewer via a Unix socket.

## Files

| File | Description |
|------|-------------|
| `internal/mcp/types.go` | MCP protocol types (JSON-RPC 2.0, tool definitions) |
| `internal/mcp/server.go` | Main MCP server with stdio transport |
| `internal/mcp/ipc.go` | Unix socket IPC for reviewer communication |
| `internal/cli/mcp.go` | `critic mcp` subcommand |
| `internal/cli/review.go` | `critic review` subcommand |

## Configuration

Add to your Claude Code MCP settings (`.claude/settings.json` or via Claude Code settings):

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

### Server Options

```
critic mcp [flags]

Flags:
  -s, --socket string      Unix socket path (default "/tmp/critic-hitl.sock")
  -t, --timeout duration   Feedback timeout (default 5m0s)
```

## Tools

### get_review_feedback

Blocks and waits for feedback from the human reviewer. Call this before completing significant changes.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "summary": {
      "type": "string",
      "description": "Brief summary of what you've done for the reviewer"
    }
  },
  "required": ["summary"]
}
```

**Behavior:**
- Notifies connected reviewers that feedback is requested
- Blocks until feedback is received or timeout (default 5 minutes)
- Returns the feedback text, prefixed with "APPROVED:" or "REJECTED:" if applicable

### notify_reviewer

Sends a notification to the reviewer without waiting for a response. Use for status updates.

**Input Schema:**
```json
{
  "type": "object",
  "properties": {
    "message": {
      "type": "string",
      "description": "The message to send to the reviewer"
    }
  },
  "required": ["message"]
}
```

## Sending Feedback

### Using the review command

```bash
# Plain feedback
critic review "Looks good, but add error handling to the API call"

# Approve the changes
critic review --approve "LGTM, ship it!"

# Reject the changes
critic review --reject "Please refactor this to use the existing helper"

# Read from stdin
echo "Your feedback" | critic review
```

### Using netcat

```bash
# Plain text (treated as feedback)
echo "Your feedback message" | nc -U /tmp/critic-hitl.sock

# JSON format for structured messages
echo '{"type":"approved","feedback":"LGTM"}' | nc -U /tmp/critic-hitl.sock
echo '{"type":"rejected","feedback":"Needs work"}' | nc -U /tmp/critic-hitl.sock
```

## Message Types

### Reviewer to Server

| Type | Description |
|------|-------------|
| `feedback` | General feedback (default for plain text) |
| `approved` | Approval with optional comment |
| `rejected` | Rejection with reason |

### Server to Reviewer

| Type | Description |
|------|-------------|
| `waiting` | Claude is waiting for feedback |
| `notification` | Status update from Claude |

## Example Workflow

1. Claude Code calls `get_review_feedback` with a summary of changes
2. MCP server notifies connected reviewers and blocks
3. Reviewer examines the changes and sends feedback:
   ```bash
   critic review --approve "Looks good, but consider adding a test"
   ```
4. MCP server returns feedback to Claude Code
5. Claude addresses feedback and may call `get_review_feedback` again

## Prompting Claude to Use HITL

Add to your `CLAUDE.md` or system prompt:

```markdown
Before completing any significant code changes, call get_review_feedback with
a summary of what you've done. Wait for reviewer approval before proceeding.
Address any feedback in subsequent iterations.
```

## Protocol Details

The MCP server implements the Model Context Protocol (MCP) version 2024-11-05 over stdio using JSON-RPC 2.0.

### Supported Methods

| Method | Description |
|--------|-------------|
| `initialize` | Initialize the server |
| `initialized` | Notification that initialization is complete |
| `tools/list` | List available tools |
| `tools/call` | Call a tool |
| `ping` | Health check |
