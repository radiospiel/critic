# Message Database Architecture

> **Note:** This document describes the database-based message storage system that replaced the socket-based human-in-the-loop interactions.

## Overview

The message database system provides persistent, threaded comment storage for human-AI code review interactions. It uses SQLite to store messages, track read status, manage conversation threads, and handle message resolution.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Interface                          │
├─────────────────────────────────────────────────────────────────┤
│  FileListModel              DiffViewModel      CommentEditor    │
│  - File highlights          - Thread rendering - Comment input  │
│  - Unread indicators        - Resolve hotkey   - Reply creation │
└────────────┬────────────────────────┬──────────────────────────┘
             │                        │
             ├────────────────────────┤
             │                        │
             v                        v
┌─────────────────────────┐  ┌──────────────────────────┐
│   FileManager           │  │     MessageDB            │
│   (.comments files)     │  │     (.critic.db)         │
│                         │  │                          │
│  - Store comment text   │  │  - Store metadata        │
│  - UUID in fence        │  │  - Thread relationships  │
│  - File persistence     │  │  - Read/Resolve status   │
└─────────────────────────┘  └──────────┬───────────────┘
                                        │
                                        v
                             ┌──────────────────────┐
                             │   MCP Server         │
                             │                      │
                             │  - Query unresolved  │
                             │  - Return threads    │
                             └──────────────────────┘
```

## Database Schema

### Messages Table

```sql
CREATE TABLE messages (
    uuid TEXT PRIMARY KEY,
    author TEXT NOT NULL CHECK(author IN ('human', 'ai')),
    status TEXT NOT NULL CHECK(status IN ('new', 'delivered', 'resolved')),
    read_status TEXT NOT NULL DEFAULT 'read' CHECK(read_status IN ('unread', 'read')),
    message TEXT NOT NULL,
    file_path TEXT NOT NULL,
    line_number INTEGER NOT NULL,
    parent_uuid TEXT,
    code_version TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (parent_uuid) REFERENCES messages(uuid) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_file_path ON messages(file_path);
CREATE INDEX idx_messages_parent_uuid ON messages(parent_uuid);
CREATE INDEX idx_messages_read_status ON messages(read_status) WHERE author = 'ai';
```

### Field Descriptions

- **uuid**: Unique identifier (UUID v4) for the message
- **author**: Either "human" or "ai" indicating who created the message
- **status**: Message lifecycle state:
  - `new`: Just created, not yet processed
  - `delivered`: Sent to/from AI
  - `resolved`: User marked as resolved (entire thread)
- **read_status**: Whether AI message has been displayed to user:
  - `unread`: AI message not yet shown (default for AI messages)
  - `read`: Message has been displayed (default for human messages)
- **message**: The actual comment text (may be multi-line)
- **file_path**: Git-relative path to the file (e.g., "src/main.go")
- **line_number**: Line number in the file (1-indexed)
- **parent_uuid**: NULL for root messages, UUID of parent for replies
- **code_version**: Git commit hash when message was created
- **created_at**: Timestamp when message was created
- **updated_at**: Timestamp when message was last modified

## Comment Format

### Standard Format

All comment blocks use the uppercase format with UUID:

```
--- CRITIC 3 LINES a1b2c3d4-e5f6-7890-abcd-ef1234567890 ---
This is the comment text
It can span multiple lines
And contain any content
--- CRITIC END ---
```

**Format Rules:**
- Opening fence: `--- CRITIC <count> LINE|LINES <uuid> ---`
- Use `LINE` for single line, `LINES` for multiple lines
- UUID must be a valid UUID v4 (lowercase hex with hyphens)
- Closing fence: `--- CRITIC END ---`
- All keywords (CRITIC, LINE/LINES, END) must be uppercase

### Thread Format (in UI)

When displaying a thread with replies:

```
--- CRITIC 5 LINES a1b2c3d4-e5f6-7890-abcd-ef1234567890 ---
Original comment from human
human> Reply from human
ai> Reply from AI
human> Another reply from human
--- CRITIC END ---
```

### Resolved Thread

```
--- CRITIC 3 LINES a1b2c3d4-e5f6-7890-abcd-ef1234567890 ---
(Resolved) Original comment from human
human> This was fixed
ai> Looks good!
--- CRITIC END ---
```

## Message Lifecycle

### 1. Human Creates Comment

```
User types comment in CommentEditor
         ↓
CommentSavedMsg event
         ↓
App creates message in DB (author=human, status=new, read_status=read)
         ↓
Comment saved to .comments file with UUID in fence
         ↓
UI refreshes to show comment
```

### 2. AI Responds to Comment

```
AI calls get_review_feedback via MCP
         ↓
MCP server queries DB for unresolved messages
         ↓
Returns first unresolved thread in diff format
         ↓
AI processes and creates reply
         ↓
Reply saved as new message (author=ai, parent_uuid=root, read_status=unread)
         ↓
Comment file updated with thread
```

### 3. User Views AI Response

```
User navigates to file with unread AI message
         ↓
File list shows RED indicator (▌)
         ↓
User opens file, DiffView renders thread
         ↓
renderCommentPreview loads thread from DB
         ↓
AI messages marked as read (read_status=unread → read)
         ↓
File list indicator changes: RED → YELLOW
```

### 4. User Resolves Thread

```
User positions cursor on comment line
         ↓
User presses 'r' hotkey
         ↓
resolveCommentAtCursor() called
         ↓
DB marks root message and all replies as resolved (status=resolved)
         ↓
UI refreshes to show "(Resolved)" suffix
```

## API Reference

### MessageDB Methods

#### Create and Update Operations

```go
// Create root message
func (db *DB) CreateMessage(
    author Author,
    message string,
    filePath string,
    lineNumber int,
    codeVersion string,
) (*Message, error)

// Create reply to existing message
func (db *DB) CreateReply(
    author Author,
    message string,
    parentUUID string,
) (*Message, error)

// Update existing message content (for edits)
func (db *DB) UpdateMessage(
    uuid string,
    newMessage string,
) error
```

**Note:** When a user edits a comment in the UI, the `UpdateMessage` function is called to update the message content in the database while preserving the UUID, timestamps, and thread relationships.

#### Query Operations

```go
// Get single message by UUID
func (db *DB) GetMessage(uuid string) (*Message, error)

// Get all messages in a thread (root + all replies)
func (db *DB) GetThreadMessages(rootUUID string) ([]*Message, error)

// Get all unresolved root messages (for MCP)
func (db *DB) GetUnresolvedRootMessages() ([]*Message, error)

// Get all root messages for a specific file
func (db *DB) GetMessagesByFile(filePath string) ([]*Message, error)

// Get all files with unread AI messages (for file list highlighting)
func (db *DB) GetFilesWithUnreadAIMessages() ([]string, error)
```

#### Update Operations

```go
// Mark entire thread as resolved
func (db *DB) MarkAsResolved(uuid string) error

// Mark AI message as read
func (db *DB) MarkAsRead(uuid string) error

// Update message status
func (db *DB) UpdateMessageStatus(uuid string, status Status) error
```

## UI Components Integration

### FileListModel

**Responsibilities:**
- Display file list with comment indicators
- Highlight files with unread AI messages (RED)
- Highlight files with regular comments (YELLOW)

**Database Usage:**
```go
// Check for unread AI messages
unreadFiles, err := m.messageDB.GetFilesWithUnreadAIMessages()
```

### DiffViewModel

**Responsibilities:**
- Render comment threads inline with code
- Display "human>" and "ai>" prefixes
- Show "(Resolved)" indicator
- Mark AI messages as read when displayed
- Handle resolve hotkey

**Database Usage:**
```go
// Load thread for display
thread, err := m.messageDB.GetThreadMessages(comment.UUID)

// Mark AI messages as read
m.messageDB.MarkAsRead(msg.UUID)

// Resolve thread
m.messageDB.MarkAsResolved(comment.UUID)
```

### CommentEditor

**Responsibilities:**
- Create new root comments
- Update existing comment content
- Create replies to existing comments (future enhancement)

**Integration in App:**
```go
// Create new comment
dbMsg, err := m.messageDB.CreateMessage(
    messagedb.AuthorHuman,
    msg.Comment,
    filePath,
    msg.LineNum,
    codeVersion,
)

// Update existing comment
err := m.messageDB.UpdateMessage(existingUUID, msg.Comment)

// Create reply (future enhancement)
dbMsg, err := m.messageDB.CreateReply(
    messagedb.AuthorHuman,
    replyText,
    parentUUID,
)
```

**Edit Flow:**
When a user edits an existing comment:
1. CommentEditor loads with existing comment text
2. User modifies and saves
3. App checks if comment has UUID
4. If UUID exists: calls `UpdateMessage()` to update content in DB
5. If no UUID: creates new message (shouldn't happen in normal flow)
6. Comment file updated with new text, preserving UUID

## MCP Server Integration

### get_review_feedback Tool

The MCP server's `get_review_feedback` tool has been modified to query the database instead of waiting for socket-based feedback.

**Flow:**

```go
func (s *Server) handleGetReviewFeedback(req Request, params CallToolParams) error {
    // 1. Query database for unresolved messages
    unresolved, err := s.messageDB.GetUnresolvedRootMessages()

    // 2. Return only the first unresolved message
    rootMsg := unresolved[0]

    // 3. Load the complete thread
    thread, err := s.messageDB.GetThreadMessages(rootMsg.UUID)

    // 4. Format as diff with thread context
    response := s.formatCommentAsDiff(rootMsg, thread)

    // 5. Return to AI
    return s.sendToolResult(req.ID, response)
}
```

**Response Format:**

```
@src/main.go 42
(code context would go here)

--- critic 4 lines a1b2c3d4-e5f6-7890-abcd-ef1234567890 ---
Original human comment about the code
human> Follow-up question from human
ai> Previous AI response
human> Human's reply to AI
--- critic end ---
```

## File Indicators

The file list uses color-coded indicators to show comment status:

| Indicator | Color | Meaning |
|-----------|-------|---------|
| ▌ | Red (#196) | File has unread AI messages |
| ▌ | Yellow (#220) | File has comments (all read) |
| (space) | - | No comments |

The indicator changes automatically:
- **RED → YELLOW**: When user views file and AI messages are marked as read
- **YELLOW → (none)**: When all comments are deleted
- **(none) → YELLOW**: When first comment is added

## Threading Model

Messages form a tree structure with one root and multiple replies:

```
Root Message (parent_uuid = NULL)
├── Reply 1 (parent_uuid = root.uuid)
├── Reply 2 (parent_uuid = root.uuid)
│   └── Reply 2.1 (parent_uuid = reply2.uuid)  [Note: nested replies are supported]
└── Reply 3 (parent_uuid = root.uuid)
```

**Current Implementation:**
- UI displays replies in chronological order (created_at ASC)
- All replies at same level (no visual nesting of sub-replies)
- Entire thread is resolved together

**Future Enhancements:**
- Visual nesting for sub-replies
- Individual message resolution
- Reply threading UI in CommentEditor

## Performance Considerations

### Indexes

The database uses several indexes to optimize common queries:

1. **idx_messages_status**: Fast lookup of unresolved messages for MCP
2. **idx_messages_file_path**: Fast lookup of comments by file
3. **idx_messages_parent_uuid**: Fast thread traversal
4. **idx_messages_read_status** (partial): Fast lookup of unread AI messages

### Caching Strategy

Currently, there is minimal caching:
- Comment text is stored in both `.comments` files (for display) and DB (for metadata)
- Thread queries happen on each render (acceptable for typical workloads)

**Future Optimizations:**
- Cache thread data in DiffViewModel between renders
- Batch mark-as-read operations
- Use WAL mode for better concurrency (already enabled)

## Migration from Socket-Based System

### Old System

```
AI calls get_review_feedback
         ↓
MCP server waits on Unix socket
         ↓
Human reviewer sends feedback via socket
         ↓
MCP server returns feedback to AI
```

**Limitations:**
- No persistence of feedback
- No threading/conversation history
- Blocking wait with timeout
- Single feedback per session

### New System

```
AI calls get_review_feedback
         ↓
MCP server queries database
         ↓
Returns unresolved comment with full thread
         ↓
AI can see conversation history
```

**Advantages:**
- Persistent storage of all interactions
- Full conversation history
- Non-blocking (immediate response)
- Multiple comments per session
- Read status tracking
- Resolution tracking

## Error Handling

### Database Errors

All database operations return errors that should be handled:

```go
// Example error handling
msg, err := db.GetMessage(uuid)
if err != nil {
    logger.Error("Failed to get message: %v", err)
    return nil
}
if msg == nil {
    // Message not found (deleted?)
    return nil
}
```

### Consistency

The system maintains consistency between `.comments` files and the database:

1. **Comment Creation**: DB message created first, then file saved
2. **Comment Update**: File updated, then DB updated (if has UUID)
3. **Comment Deletion**: File deleted, DB message remains (orphaned)

**Note:** Currently, deleting a comment from the file doesn't delete it from the database. This is intentional to preserve conversation history.

## Testing

The message database includes comprehensive tests:

```bash
go test ./internal/messagedb -v
```

**Test Coverage:**
- Creating root messages and replies
- Querying messages and threads
- Marking messages as resolved/read
- Retrieving unresolved messages
- File-based queries
- Edge cases (missing messages, empty threads)

## Future Enhancements

1. **Reply UI**: Add interface for creating replies directly (not just root comments)
2. **Message Editing**: Track edit history in database
3. **Message Deletion**: Soft delete with timestamps
4. **Search**: Full-text search across all messages
5. **Export**: Export conversations to markdown/JSON
6. **Analytics**: Track resolution time, response patterns
7. **Notifications**: Desktop notifications for new AI messages
8. **Sync**: Multi-user support with conflict resolution

## Related Documentation

- [MCP Server](../mcp.md) - MCP protocol integration
- [CLI](../CLI.md) - Command-line interface
- [File Watcher](./file-watcher.md) - File change detection
