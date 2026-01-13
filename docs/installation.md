# Installation

## Requirements

- Go 1.24 or later
- Git
- A terminal with ANSI color support

## Building from Source

```bash
# Clone the repository
git clone git@git.15b.it:eno/critic.git
cd critic

# Build
go build -o critic ./cmd/critic

# Install to $GOPATH/bin
go install ./cmd/critic
```

## Verifying Installation

```bash
# Check version
critic --help

# Run in a git repository
cd /path/to/your/repo
critic
```

## MCP Server Configuration

To use Critic with Claude Code for human-in-the-loop reviews, add to your MCP settings:

**Global settings** (`~/.claude/settings.json`):
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

**Project settings** (`.claude/settings.json` in repo):
```json
{
  "mcpServers": {
    "critic": {
      "command": "/path/to/critic",
      "args": ["mcp"]
    }
  }
}
```

## Prompting Claude to Use Critic

Add to your `CLAUDE.md`:

```markdown
Before completing any significant code changes, call get_review_feedback with
a summary of what you've done. Wait for reviewer approval before proceeding.
Address any feedback in subsequent iterations.
```

## Database Location

Critic stores comments in `.critic.db` at the git repository root. This file:
- Uses SQLite with WAL mode
- Can be committed to version control (optional)
- Is automatically created on first comment

## Logging

Debug logs are written to `/tmp/critic.log`. Control verbosity:

```bash
# Enable debug logging
CRITIC_LOG_LEVEL=DEBUG critic

# Available levels: DEBUG, INFO, WARN, ERROR
```

## Terminal Compatibility

Critic works best with terminals that support:
- 256 colors or true color
- Unicode characters
- Alternate screen buffer

Tested terminals:
- iTerm2 (macOS)
- Terminal.app (macOS)
- Alacritty
- Kitty
- Windows Terminal

## Troubleshooting

### "Not a git repository"

Critic must be run from within a git repository:
```bash
cd /path/to/git/repo
critic
```

### Database errors

If you encounter SQLite errors, try removing the database:
```bash
rm .critic.db
```

### File watcher issues

If file changes aren't detected, check:
- File descriptor limits (`ulimit -n`)
- That you're diffing against `current` (working directory)

Increase file descriptor limit if needed:
```bash
ulimit -n 10240
```
