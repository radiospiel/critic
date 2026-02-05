# Architecture

## Directory Structure

```
critic/
├── cmd/critic/           # Entry point
│   └── main.go
├── internal/             # Core application (not exported)
│   ├── app/              # Application state machine
│   ├── cli/              # CLI parsing (Cobra)
│   ├── config/           # Configuration (extensions, etc.)
│   ├── git/              # Git operations
│   ├── highlight/        # Syntax highlighting
│   ├── mcp/              # MCP server
│   ├── messagedb/        # SQLite message storage
│   └── ui/               # UI components
├── pkg/                  # Public packages
│   ├── critic/           # Core interfaces (Messaging)
│   └── types/            # Shared types (Diff, Comment)
├── simple-go/            # Utility packages
│   ├── assert/           # Test assertions
│   ├── dump/             # Debug printing
│   ├── logger/           # File logging
│   ├── must/             # Panic-on-error helpers
│   ├── preconditions/    # Input validation
│   └── utils/            # General utilities
├── teapot/               # Custom UI framework
└── tests/                # Integration tests
```

## Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Terminal UI (Bubble Tea)                  │
├─────────────────────────────────────────────────────────────┤
│ ┌──────────────────┐       ┌──────────────────┐             │
│ │  FileListWidget  │       │   DiffViewModel  │             │
│ │   (left pane)    │       │   (right pane)   │             │
│ └──────────────────┘       └──────────────────┘             │
│ ┌──────────────────────────────────────────────┐            │
│ │          CommentEditor (modal)               │            │
│ └──────────────────────────────────────────────┘            │
├─────────────────────────────────────────────────────────────┤
│             Application Model (internal/app/)                │
│  - Event loop, state management, business logic              │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────────┐  ┌──────────────────┐  ┌──────────────┐ │
│ │  Git Watcher    │  │  Base Resolver   │  │ Diff Engine  │ │
│ │ (file changes)  │  │  (polls refs)    │  │ (parse diff) │ │
│ └─────────────────┘  └──────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│ ┌──────────────────────────────────────────────┐            │
│ │   Message Database (.critic.db)              │            │
│ │   Implements: critic.Messaging interface     │            │
│ └──────────────────────────────────────────────┘            │
├─────────────────────────────────────────────────────────────┤
│ ┌──────────────────┐  ┌──────────────────┐                  │
│ │   Git Commands   │  │   MCP Server     │                  │
│ │   (diff, refs)   │  │ (Claude HITL)    │                  │
│ └──────────────────┘  └──────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
```

## Core Data Structures

### Diff Model (`pkg/types/diff.go`)

```go
type Diff struct {
    Files []*FileDiff
}

type FileDiff struct {
    OldPath, NewPath     string
    OldMode, NewMode     string
    IsNew, IsDeleted     bool
    IsRenamed, IsBinary  bool
    Hunks                []*Hunk
}

type Hunk struct {
    OldStart, OldLines   int
    NewStart, NewLines   int
    Header               string
    Lines                []*Line
}

type Line struct {
    Type     LineType  // LineContext, LineAdded, LineDeleted
    Content  string
    OldNum   int
    NewNum   int
}
```

### Message Model (`pkg/critic/messaging.go`)

```go
type Conversation struct {
    UUID        string
    Status      string      // "new", "delivered", "resolved"
    FilePath    string
    LineNumber  int
    CodeVersion string
    Messages    []Message
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Message struct {
    UUID      string
    Author    Author      // AuthorHuman, AuthorAI
    Message   string
    IsUnread  bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

## Key Interfaces

### Messaging Interface (`pkg/critic/messaging.go`)

The `Messaging` interface abstracts comment storage:

```go
type Messaging interface {
    GetConversations(status string) ([]Conversation, error)
    GetConversationsByFile(filePath string) ([]Conversation, error)
    GetFullConversation(uuid string) (*Conversation, error)
    ReplyToConversation(uuid, message string, author Author) (*Message, error)
    CreateConversation(author Author, message, filePath string,
                       lineNumber int, codeVersion, context string) (*Conversation, error)
    MarkConversationAs(conversationUUID string, update ConversationUpdate) error
    MarkMessageAs(messageUUID string, status MessageReadStatus) error
    Close() error
}
```

Implemented by `internal/messagedb.DB` using SQLite.

## Database Schema

Comments are stored in `.critic.db` at the git root.

```sql
CREATE TABLE messages (
    id TEXT PRIMARY KEY,                    -- UUID v7
    author TEXT NOT NULL,                   -- 'human' or 'ai'
    status TEXT NOT NULL,                   -- 'new', 'delivered', 'resolved'
    read_status TEXT NOT NULL DEFAULT 'read',
    message TEXT NOT NULL,
    file_path TEXT NOT NULL,
    lineno INTEGER NOT NULL,
    conversation_id TEXT NOT NULL,          -- FK to root message
    sha1 TEXT NOT NULL,                     -- git commit hash
    context TEXT,                           -- surrounding code
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (conversation_id) REFERENCES messages(id)
);

-- Indexes
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_file_path ON messages(file_path);
CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_read_status ON messages(read_status);
```

**Threading model:** Root messages have `conversation_id = id`. Replies have `conversation_id` pointing to the root message.

## Git Integration (`internal/git/`)

### Diff Retrieval

```go
// Get diff between two refs
func GetDiffBetween(base, target string, paths []string) (*types.Diff, error)
```

Supports:
- Merge-base resolution
- Branch/tag/commit references
- Path filtering
- Whitespace options

### File Watcher (`internal/git/watcher.go`)

Three-stage pipeline for efficient file change detection:

```
fsnotify.Event → eventLoop → filterLoop → debounceLoop → FileChange
```

- **eventLoop**: Receives raw fsnotify events
- **filterLoop**: Filters by configured paths
- **debounceLoop**: Per-file debouncing (default 100ms)

### Base Resolver (`internal/git/baseresolver.go`)

Polls git refs every 10 seconds to detect changes:
- Merge-base with main/master
- Remote branch updates
- Local branch updates

Triggers callbacks when bases change.

## UI Framework (`teapot/`)

Custom widget system built on [Bubble Tea](https://github.com/charmbracelet/bubbletea):

- `Widget` interface - Base contract for all UI elements
- `SelectableList[T]` - File list with selection
- `DiffViewModel` - Diff viewing with viewport
- `CommentEditor` - Text input for comments
- `Buffer`/`SubBuffer` - Off-screen rendering
- `Compositor` - Layout management

## MCP Server (`internal/mcp/`)

JSON-RPC 2.0 server over stdin/stdout implementing MCP protocol v2024-11-05.

**Tools exposed:**
- `get_critic_conversations` - List conversations by status
- `get_full_critic_conversation` - Get conversation with all messages
- `reply_to_critic_conversation` - AI adds reply

## Testing

### Test Framework

Uses Go's `testing` package with custom assertions from `simple-go/assert/`:

```go
assert.Equals(t, actual, expected, "description")
assert.Contains(t, slice, item)
assert.True(t, condition)
assert.Nil(t, value)
assert.Panics(t, func() { ... })
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/messagedb -v
go test ./internal/git -v

# Run with coverage
go test -cover ./...
```

### Test Organization

| Package | Coverage |
|---------|----------|
| `internal/messagedb` | Database operations, threading |
| `internal/git` | Diff parsing, base resolution, watcher |
| `internal/cli` | CLI argument parsing |
| `internal/highlight` | Syntax highlighting |
| `teapot/` | UI framework components |

### Writing Tests

Follow project conventions:
- Use `assert` package instead of manual `if` checks
- Use descriptive variable names: `actual`, `expected`
- Include expected values in test cases
- Clean up resources (temp files, databases)

Example:
```go
func TestCreateConversation(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    conv, err := db.CreateConversation(
        critic.AuthorHuman,
        "This needs refactoring",
        "main.go",
        42,
        "abc123",
        "func main() {",
    )

    assert.Nil(t, err)
    assert.NotNil(t, conv)
    assert.Equals(t, conv.FilePath, "main.go")
    assert.Equals(t, conv.LineNumber, 42)
}
```

## Logging

File-based logging to `/tmp/critic.log`:

```go
logger.Debug("message: %v", value)
logger.Info("message")
logger.Warn("message")
logger.Error("message: %v", err)
```

Set log level via environment:
```bash
CRITIC_LOG_LEVEL=DEBUG critic
```
