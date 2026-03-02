# System Design

This document describes the communication architecture between the WebUI, agent CLI, and other components in Critic.

## Overview

Critic is a web server which orchestrates the communication between a coding agent and a human user. It allows humans to review code changes made by the agent, and to ask for adjustments, without going through GitHub. This allows for a faster turn-around between changes and review, allowing the agent to actively employ a human-in-the-loop review process.

Critic uses a **loosely-coupled, event-driven architecture**. This design allows the agent CLI to run independently from the HTTP server. Messaging between the agent and the user takes place using a SQLite database, changes in the source code are directly handled via a git-controlled file system.

The event-driven architecture allows the WebUI to automatically reflects changes, both to the source files and in the messages between users and agents, in real-time. The WebUI is also only *one* UI – a terminal-driven UI would equally possible.

**Note that critic is designed as a single-user experience.** A typical scenario runs the critic web server on the same machine that also runs the coding agent, and only listens on localhost. It is important to not deploy critic in an unsecured environment, since the web client has access to **the entire tree of source files**.

## Architecture Diagram

<img width="2445" height="2010" alt="image" src="https://github.com/user-attachments/assets/1554099e-b90a-422d-9e6d-22d4739589da" />

## Components

### 1. Agent CLI (`src/cli/agent.go`)

The agent CLI provides commands for AI agents to interact with code review conversations:

| Command | Description |
|---------|-------------|
| `critic agent conversations` | List conversations (uuid, author, status) |
| `critic agent conversation <uuid>` | Get a full conversation thread |
| `critic agent reply <uuid> <msg>` | Reply to a conversation |
| `critic agent announce <msg>` | Post an announcement |
| `critic agent explain <file> <line> <msg>` | Post an explanation |

The agent CLI connects directly to the SQLite database to read and write messages.

### 2. HTTP Server (`src/api/server/`)

The HTTP server provides:
- HTTP/Connect API for the frontend
- WebSocket hub for real-time updates
- Embedded react frontend

The HTTP server employs Git and database watchers to detect changes, and informs the frontend via Websockets about changes.

#### Database Watcher (`src/messagedb/db_watcher.go`)

Detects database changes made by external processes (like the agent CLI).

**How it works:**
- Polls the `_db_mtime` table every 1000ms. Note that this is using fresh connections, to work around limitations with WAL mode.
- SQLite triggers automatically update `_db_mtime` when messages change
- When a change is detected, notifies the API server

#### Git Watcher (`src/git/git_watcher.go`)

Monitors the `.git/` directory for changes using fsnotify.

**Features:**
- Debounced notifications (100ms default) to prevent notification spam
- Detects commits, checkouts, pulls, and other git operations

#### WebSocket Hub (`src/webui/websocket.go`)

Manages WebSocket connections and broadcasts updates to all connected clients.

**Message types:**
- `{"type":"reload"}` - Database changed, refresh data
- `{"type":"file-changed","path":"..."}` - Watched file changed
- `{"type":"git-changed"}` - Git state changed

## Communication Flow

### When Agent CLI Adds a Comment

```
1. AI Agent (via CLI)
   └─> critic agent reply <uuid> "message"
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

5. Agent CLI
   └─> On next `critic agent conversations` call, sees the new message
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

SQLite doesn't support cross-process change notifications. Polling the `_db_mtime` table with fresh connections is a reliable way to detect changes made by any process (agent CLI, web UI, etc.).

### Why Triggers for `_db_mtime`?

Using triggers ensures the modification time is always updated, regardless of which process or code path modifies the data. This provides a single source of truth for "did anything change?"

### Why Fresh Connections for DBWatcher?

SQLite connections can cache data, especially in WAL mode. Opening a fresh connection for each poll guarantees we see the latest committed changes from other processes.

### Why WebSocket Instead of Server-Sent Events?

WebSocket provides bidirectional communication, allowing the frontend to send file watch requests and receive updates on the same connection.
