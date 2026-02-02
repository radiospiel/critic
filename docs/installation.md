# Installation

## Requirements

- Go 1.24 or later
- Git
- Node.js (for frontend development)

## Building from Source

```bash
git clone git@git.15b.it:eno/critic.git
cd critic
go build -o critic ./src/cmd/
```

To install globally:

```bash
go install ./src/cmd/
```

## MCP Server Configuration

To integrate Critic with Claude Code for AI-assisted code review:

```bash
claude mcp add critic -- /path/to/critic mcp
```

This enables Claude to read and respond to code review comments.

## Database

Comments are stored in `.critic/critic.db` (SQLite) at the git repository root. This directory must not be shared and should be added to `.gitignore`.

## Logging

Debug logs are written to `/tmp/critic.log`. Control log level with the environment variable:

```bash
CRITIC_LOG_LEVEL=DEBUG critic
```
