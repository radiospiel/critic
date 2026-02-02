# System Design

This document describes the communication architecture between the WebUI, MCP server, and other components in Critic.

## Overview

Critic uses a **loosely-coupled, event-driven architecture** where components communicate indirectly through:

1. A shared SQLite database
2. A git file system watcher
3. A database modification time watcher
4. WebSocket broadcasts to connected clients

This design allows the MCP server to run independently (even in a different process) while the WebUI automatically reflects changes in real-time.

## Architecture Diagram

blob:https://whimsical.com/e1f58a5f-c1aa-4957-87c0-ce27e0e184aa<img width="1219" height="968" alt="image" src="https://github.com/user-attachments/assets/4169ba12-2e6d-441b-b540-c7a9dc5a9439" />

```
┌─────────────────────┐         ┌─────────────────────┐
│    Claude Code      │         │       Browser       │
│    (AI Client)      │         │      (WebUI)        │
└──────────┬──────────┘         └──────────┬──────────┘
           │ JSON-RPC 2.0                  │ WebSocket
           │ (stdin/stdout)                │
           ▼                               ▼
┌──────────────────────┐       ┌───────────────────────┐
│     MCP Server       │       │      API Server       │
│  (src/mcp/server.go) │       │ (src/api/server/)     │
└──────────┬───────────┘       └───────────┬───────────┘
           │                               │
           │   ┌───────────────────────────┤
           │   │                           │
           │   │  ┌──────────────────┐     │
           │   │  │    DBWatcher     │     │
           │   │  │ (polls _db_mtime)│◄────┤
           │   │  └────────┬─────────┘     │
           │   │           │               │
           ▼   ▼           ▼               ▼
    ┌────────────────────────────────────────────┐
    │                SQLite Database              │
    │            (.critic/critic.db)              │
    │  ┌──────────────────────────────────────┐  │
    │  │  messages table                       │  │
    │  │  _db_mtime table (with triggers)      │  │
    │  └──────────────────────────────────────┘  │
    └────────────────────────────────────────────┘

┌────────────────────────────────────────────────┐
│              Git Repository                     │
│  ┌──────────────────────────────────────────┐  │
│  │  .git/ directory                          │  │
│  │    └── GitWatcher (fsnotify)              │  │
│  └──────────────────────────────────────────┘  │
└────────────────────────────────────────────────┘
```

## Components

### 1. MCP Server (`src/mcp/`)

The MCP server provides tools for AI assistants to interact with code review conversations:

| Tool | Description |
|------|-------------|
| `get_critic_conversations` | List all conversation UUIDs |
| `get_full_critic_conversation` | Get a full conversation thread |
| `reply_to_critic_conversation` | Add a reply to a conversation |

The MCP server connects directly to the SQLite database to read and write messages.

### 2. API Server (`src/api/server/`)

The API server provides:
- HTTP/Connect API for the frontend
- WebSocket hub for real-time updates
- Git and database watchers for change detection

### 3. Database Watcher (`src/messagedb/db_watcher.go`)

Detects database changes made by external processes (like the MCP server).

**How it works:**
- Polls the `_db_mtime` table every 1000ms using fresh connections
- SQLite triggers automatically update `_db_mtime` when messages change
- When a change is detected, notifies the API server

**Why fresh connections?** Using a fresh connection each poll ensures changes made by other processes (like the MCP server) are visible, even in SQLite's WAL mode.

### 4. Git Watcher (`src/git/git_watcher.go`)

Monitors the `.git/` directory for changes using fsnotify.

**Features:**
- Debounced notifications (100ms default) to prevent notification spam
- Detects commits, checkouts, pulls, and other git operations

### 5. WebSocket Hub (`src/webui/websocket.go`)

Manages WebSocket connections and broadcasts updates to all connected clients.

**Message types:**
- `{"type":"reload"}` - Database changed, refresh data
- `{"type":"file-changed","path":"..."}` - Watched file changed
- `{"type":"git-changed"}` - Git state changed

## Communication Flow

### When MCP Server Adds a Comment

```
1. Claude AI (via MCP)
   └─> MCP.handleReplyToCriticConversation()
       └─> db.CreateReply()
           └─> SQL: INSERT INTO messages (...)

2. SQLite Database
   └─> AFTER INSERT Trigger fires
       └─> UPDATE _db_mtime SET mtime_msec = ...

3. API Server (DBWatcher polling)
   └─> checkMtime() detects change
       └─> Sends notification on changesChan

4. API Server (handleDBChanges)
   └─> Receives notification
       └─> wsHub.Broadcast({"type":"reload"})

5. WebUI (Browser)
   └─> Receives {"type":"reload"}
       └─> Fetches updated data via API
```

### When User Adds a Comment in WebUI

```
1. WebUI (Browser)
   └─> POST /api/comments
       └─> API Server receives request

2. API Server
   └─> db.CreateConversation() or db.CreateReply()
       └─> SQL: INSERT INTO messages (...)

3. SQLite Database
   └─> AFTER INSERT Trigger fires
       └─> UPDATE _db_mtime SET mtime_msec = ...

4. DBWatcher (same server, sees change immediately)
   └─> Broadcasts to all OTHER connected clients

5. MCP Server
   └─> On next tool call, sees the new message
```

### When Git State Changes

```
1. User runs git command (commit, checkout, etc.)
   └─> Files in .git/ are modified

2. GitWatcher (fsnotify)
   └─> Receives file system events
       └─> Debounces for 100ms

3. GitWatcher
   └─> Sends notification on changesChan

4. API Server (handleGitChanges)
   └─> wsHub.Broadcast({"type":"git-changed"})

5. WebUI (Browser)
   └─> Receives {"type":"git-changed"}
       └─> Refreshes diff data
```

## Database Schema (Relevant Parts)

### `_db_mtime` Table

```sql
CREATE TABLE _db_mtime (
    tablename TEXT PRIMARY KEY,
    mtime_msec INTEGER
);
```

### Message Change Triggers

```sql
CREATE TRIGGER _messages_insert_mtime AFTER INSERT ON messages
BEGIN
    UPDATE _db_mtime
    SET mtime_msec = CAST(unixepoch('subsec') * 1000 AS INTEGER)
    WHERE tablename = 'messages';
END;

CREATE TRIGGER _messages_update_mtime AFTER UPDATE ON messages ...
CREATE TRIGGER _messages_delete_mtime AFTER DELETE ON messages ...
```

## Timing and Performance

| Component | Interval | Notes |
|-----------|----------|-------|
| DBWatcher | 1000ms polling | Balance between responsiveness and CPU usage |
| GitWatcher | 100ms debounce | Prevents notification spam from rapid changes |
| WebSocket ping | 30s | Keeps connection alive |

## Design Decisions

### Why Polling Instead of Notifications?

SQLite doesn't support cross-process change notifications. Polling the `_db_mtime` table with fresh connections is a reliable way to detect changes made by any process (MCP server, CLI, etc.).

### Why Triggers for `_db_mtime`?

Using triggers ensures the modification time is always updated, regardless of which process or code path modifies the data. This provides a single source of truth for "did anything change?"

### Why Fresh Connections for DBWatcher?

SQLite connections can cache data, especially in WAL mode. Opening a fresh connection for each poll guarantees we see the latest committed changes from other processes.

### Why WebSocket Instead of Server-Sent Events?

WebSocket provides bidirectional communication, allowing the frontend to send file watch requests and receive updates on the same connection.
