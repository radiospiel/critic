# Usage

## Command Line

### Web UI

```bash
critic webui                         # Start web UI on default port
critic webui --port=8080             # Start on specific port
```

### Agent CLI

```bash
critic agent conversations                           # List all conversations (JSON)
critic agent conversations --status=actionable       # List actionable conversations
critic agent conversation <uuid>                     # Show full conversation
critic agent reply <uuid> "message"                  # Reply to a conversation
critic agent announce "message"                      # Post an announcement
critic agent explain <file> <line> "comment"         # Post an explanation
```

## Web UI Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `?` | Toggle help overlay |
| `j` / `k` | Navigate in file list |
| `Tab` | Switch focus between panes |

## Agent CLI Commands

AI agents interact with critic via `critic agent` subcommands. All output is JSON.

| Command | Description |
|---------|-------------|
| `critic agent conversations` | List conversations (uuid, last author, status) |
| `critic agent conversation <uuid>` | Get full conversation with all messages |
| `critic agent reply <uuid> <msg>` | Reply to a conversation as AI |
| `critic agent announce <msg>` | Post an announcement (marks root conversation unresolved) |
| `critic agent explain <file> <line> <msg>` | Post an explanation on a code line |

### Example Agent Workflow

1. Human reviewer opens `critic webui` and adds inline comments on the diff
2. AI agent runs `critic agent conversations --status=actionable` to discover new comments
3. AI runs `critic agent conversation <uuid>` to read the full conversation
4. AI addresses the feedback and runs `critic agent reply <uuid> "message"` to respond
5. Human sees AI response in the Web UI (auto-refreshes via WebSocket)
6. Conversation continues until human marks it resolved

### Filters

**Status filter** (`--status`): `unresolved`, `resolved`, `actionable` (comma-separated)

```bash
critic agent conversations --status=actionable
critic agent conversations --status=unresolved,resolved
```

**Last-author filter** (`--last-author`): `human`, `ai` (comma-separated)

```bash
critic agent conversations --last-author=human
```
