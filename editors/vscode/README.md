# Critic VS Code Extension

Inline code review inside VS Code, powered by [critic](https://github.com/radiospiel/critic).

## Features

- Inline comment threads reflecting critic conversations, live in your editor
- Create new review comments directly from the gutter
- Reply to, resolve, and archive conversations without leaving VS Code
- Status bar item showing connection state and unread AI reply count
- Auto-refreshes every 5 seconds (configurable)

## Requirements

The critic server must be running locally:

```bash
critic httpd
```

## Install

Download the `.vsix` directly from the running critic server:

```bash
curl -O http://localhost:65432/download/critic-vscode.vsix
code --install-extension critic-vscode.vsix
```

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `critic.serverUrl` | `http://localhost:65432` | URL of the critic server |
| `critic.pollIntervalMs` | `5000` | Poll interval in ms |
| `critic.showResolvedComments` | `false` | Show resolved conversations |

## Commands

- **Critic: Refresh Comments** — manually trigger a refresh
- **Critic: Open Web UI in Browser** — open the critic web interface
- **Critic: Resolve Conversation** — resolve the focused comment thread
- **Critic: Archive Conversation** — archive the focused comment thread
