# Web UI Architecture

This document describes the architecture of Critic's web-based user interface.

## Overview

The Web UI provides a browser-based interface for viewing git diffs and managing code review conversations. It replicates the functionality of the TUI (Terminal User Interface) while providing a more accessible experience for users who prefer graphical interfaces.

## Starting the Web UI

```bash
critic webui                    # Start on default port 8080
critic webui --port=3000        # Start on custom port
critic webui main               # Compare against main branch
critic webui -- src tests       # Only show changes in specific paths
```

## Architecture

### Technology Stack

- **Backend**: Go (Golang) standard library HTTP server
- **Frontend**: HTML + CSS + JavaScript
- **Interactivity**: [htmx](https://htmx.org/) for declarative AJAX
- **Real-time Updates**: WebSockets for push notifications
- **Styling**: CSS variables for light/dark theme support

### Package Structure

```
internal/webui/
├── server.go       # HTTP server setup and configuration
├── handlers.go     # HTTP request handlers
├── websocket.go    # WebSocket hub and client management
├── templates/      # HTML templates (embedded)
│   ├── index.html       # Main page layout
│   ├── file.html        # Single file view
│   ├── filelist.html    # File list partial
│   ├── diff.html        # Diff view partial
│   ├── conversation.html    # Single conversation
│   └── conversations.html   # Conversations list
└── static/         # Static assets (embedded)
    └── style.css   # Styles with light/dark theme
```

### Components

#### HTTP Server (`server.go`)

The server is responsible for:
- Serving static assets and HTML pages
- Loading and caching git diff data
- Managing the WebSocket hub
- Interfacing with the message database

```go
type Server struct {
    config    Config
    templates *template.Template
    messaging critic.Messaging  // Shared with TUI
    hub       *Hub              // WebSocket connections
    diff      *types.Diff       // Cached diff data
}
```

#### HTTP Handlers (`handlers.go`)

Routes and their purposes:

| Route | Method | Description |
|-------|--------|-------------|
| `/` | GET | Main page with file list and diff view |
| `/file/{path}` | GET | View specific file diff |
| `/api/files` | GET | Get file list (htmx partial) |
| `/api/diff/{path}` | GET | Get diff for file (htmx partial) |
| `/api/conversations/{path}` | GET | Get conversations for file |
| `/api/comment` | POST | Create new comment |
| `/api/reply` | POST | Reply to conversation |
| `/api/resolve/{uuid}` | POST | Mark conversation resolved |
| `/api/unresolve/{uuid}` | POST | Mark conversation unresolved |
| `/ws` | GET | WebSocket connection |
| `/static/*` | GET | Static assets |

#### WebSocket Hub (`websocket.go`)

The WebSocket system uses a hub-and-spoke pattern:

```
┌─────────────────────────────────────────┐
│                Hub                       │
│  ┌─────────────────────────────────┐    │
│  │        broadcast channel        │    │
│  └─────────────────────────────────┘    │
│           │         │         │          │
│      ┌────┴────┐┌───┴───┐┌───┴────┐     │
│      │Client 1 ││Client 2││Client 3│     │
│      └─────────┘└────────┘└────────┘     │
└─────────────────────────────────────────┘
```

- **Hub**: Central manager for all WebSocket connections
- **Clients**: Individual browser connections
- **Broadcast**: Server pushes updates to all clients

Updates are notification-only: the WebSocket message tells clients *what* changed, and clients fetch the new data via htmx requests.

### Data Flow

#### Viewing a Diff

```
1. User opens browser to /
2. Server renders index.html with file list
3. User clicks a file
4. htmx sends GET /api/diff/{path}
5. Server renders diff.html partial
6. htmx swaps content into #diff-view
```

#### Adding a Comment

```
1. User clicks comment button on a diff line
2. Modal opens with comment form
3. User submits form
4. htmx sends POST /api/comment
5. Server:
   a. Creates conversation in database
   b. Broadcasts "conversation" update via WebSocket
   c. Returns conversation.html partial
6. htmx appends new conversation to list
7. Other clients receive WebSocket message
8. Other clients refresh their views
```

#### Real-time Updates

```
┌─────────┐    WebSocket     ┌─────────┐
│ Client A│◄────────────────►│ Server  │
└────┬────┘                  └────┬────┘
     │                            │
     │ POST /api/comment          │
     │───────────────────────────►│
     │                            │
     │    {"type":"conversation"} │
     │◄───────────────────────────│
     │                            │
┌────┴────┐                       │
│ Client B│◄──────────────────────┤
└─────────┘  {"type":"conversation"}
```

### Theme System

The UI supports light and dark themes using CSS variables:

```css
:root {
    /* Light theme defaults */
    --bg-primary: #ffffff;
    --text-primary: #24292f;
    /* ... */
}

[data-theme="dark"] {
    --bg-primary: #0d1117;
    --text-primary: #c9d1d9;
    /* ... */
}
```

Theme preference is:
1. Stored in `localStorage`
2. Applied via `data-theme` attribute on `<html>`
3. Toggled with the theme button in the header

### Template System

Templates are embedded in the binary using Go's `embed` package:

```go
//go:embed templates/*.html static/*
var embeddedFS embed.FS
```

This means:
- No external files needed at runtime
- Single binary deployment
- Templates parsed at startup

### Shared Infrastructure

The Web UI shares the same infrastructure as the TUI:

- **Message Database**: SQLite database (`.critic.db`) for conversations
- **Git Operations**: Same `internal/git` package for diff generation
- **Messaging Interface**: `critic.Messaging` for conversation management

This ensures consistency between TUI and Web UI:
- Conversations created in TUI appear in Web UI
- Both interfaces see the same diff data
- Real-time updates work across both interfaces (when using MCP server)

## Security Considerations

The Web UI is designed for local development use:

- **No authentication**: Runs on localhost only
- **No CORS**: Same-origin requests only
- **No rate limiting**: Trusted local environment
- **File access**: Has full access to git repository

For production or remote access, additional security measures would be needed.

## Implemented Features

The following features have been implemented:

- **Syntax highlighting**: Uses Chroma library with CSS classes for highlighting
- **Keyboard navigation**: `j`/`k` for file list, `?` for help, `Tab` for pane switching
- **Theme toggle**: Dark/light mode with localStorage persistence
- **Local assets**: All JS (htmx) served locally, no CDN dependencies

## Testing

### End-to-End Tests

The Web UI includes Puppeteer-based e2e tests in `tests/e2e/`:

```bash
# Install dependencies
cd tests/e2e
npm install

# Run tests (requires critic binary)
npm test
```

**Test Coverage:**

| Test Category | What's Tested |
|--------------|---------------|
| Page Load | Title, header, theme toggle button |
| File List | Container exists, files load, item structure |
| Diff View | Click loads diff, line numbers, syntax highlighting |
| Theme Toggle | Default dark, toggle to light/dark, localStorage |
| Keyboard | Help overlay (?), file navigation (j/k) |
| API Endpoints | /api/files, /api/diff/{path} |
| WebSocket | htmx ws extension loaded |

### Manual Testing

To manually test the Web UI:

```bash
# Build and start
go build -o critic ./cmd/critic
./critic webui --port=8080

# Open browser
open http://localhost:8080
```

Verify:
1. File list loads with changed files
2. Clicking a file shows the diff
3. Theme toggle switches between light/dark
4. Comment buttons appear on diff lines
5. Help overlay shows on `?` key

## Future Improvements

Potential enhancements:
- Split-view for side-by-side diff
- File tree view with folder collapsing
- Search functionality
- Comment threading UI improvements
- Mobile-responsive design
