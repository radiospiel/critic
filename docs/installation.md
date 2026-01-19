# Installation

## Requirements

- Go 1.24 or later
- Git
- Terminal with ANSI color support (iTerm2, Alacritty, Kitty, Windows Terminal)

## Building from Source

```bash
git clone git@git.15b.it:eno/critic.git
cd critic
go build -o critic ./cmd/critic
```

To install globally:

```bash
go install ./cmd/critic
```

## MCP Server Configuration

To integrate Critic with Claude Code for AI-assisted code review:

```bash
claude mcp add critic -- /path/to/critic mcp
```

This enables Claude to read and respond to code review comments.

## Database

Comments are stored in `.critic.db` (SQLite) at the git repository root. This file can be committed to share comments with collaborators or added to `.gitignore` for local-only comments.

## Logging

Debug logs are written to `/tmp/critic.log`. Control log level with the environment variable:

```bash
CRITIC_LOG_LEVEL=DEBUG critic
```
