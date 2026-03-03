# Installation

## Requirements

- Go 1.24 or later
- Git
- Node.js (for frontend)
- protoc (Protocol Buffers compiler)

## Building from Source

```bash
git clone https://github.com/radiospiel/critic.git
cd critic
make build
```

To install system-wide (to `/usr/local/bin`):

```bash
make install
```

### Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build everything (deps, proto, server, frontend) |
| `make install` | Build and install to /usr/local/bin |
| `make tests` | Run unit and integration tests |
| `make clean` | Remove build artifacts |

## Agent CLI

AI coding agents interact with critic via the `critic agent` CLI subcommand. All output is JSON.

### Available Commands

| Command | Description |
|---------|-------------|
| `critic agent conversations [--status=...] [--last-author=...]` | List conversations (JSON array of uuid/author/status) |
| `critic agent conversation <uuid>` | Get full conversation with all messages |
| `critic agent reply <uuid> <message>` | Reply to a conversation (as AI) |
| `critic agent announce <message>` | Post an announcement visible in the Critic UI |
| `critic agent explain <file> <line> <comment>` | Post an explanation on a code line |

### Status Filters

The `--status` flag accepts comma-separated values:
- `unresolved` — conversations not yet resolved
- `resolved` — resolved conversations
- `actionable` — unresolved conversations where the last message is from a human (i.e., awaiting agent response)

The `--last-author` flag filters by who posted the last message: `human` or `ai`.

### Verifying Installation

```bash
# Should print [] (empty JSON array) if no conversations exist
critic agent conversations
```

## Configuring AI Agents for HITL Workflow

To enable human-in-the-loop code review, configure your AI coding agent to check for and respond to critic feedback. Add the following to your project's `CLAUDE.md` or `AGENTS.md`:

### Example CLAUDE.md Configuration

```markdown
### Human-in-the-Loop Code Review

Before committing any significant code changes, check for reviewer feedback:

- Run `critic agent conversations --status=actionable` to find conversations needing attention.
- If there are actionable conversations, read each via `critic agent conversation <uuid>`, address the feedback, and reply via `critic agent reply <uuid> '<message>'`.
- Wait for reviewer approval before proceeding. Address any feedback in subsequent iterations.
```

### Workflow Example

1. **Developer starts AI coding session** with critic installed
2. **AI implements feature**, developer reviews in `critic webui`
3. **Human reviewer** adds inline comments on the diff
4. **AI checks feedback** via `critic agent conversations --status=actionable`
5. **AI reads feedback** via `critic agent conversation <uuid>`, addresses it, and replies via `critic agent reply <uuid> '<message>'`
6. **Human approves** or continues the conversation
7. **AI commits** once approved

### Interactive Review Session

For a more interactive workflow, the human can:

1. Run `critic webui` in a terminal
2. Tell the AI to wait for feedback
3. Add comments in the web UI
4. Press Enter in Claude Code to signal "feedback ready"
5. AI processes all feedback and responds

## Database

Comments are stored in `.critic/critic.db` (SQLite) at the git repository root. This directory must not be shared and should be added to `.gitignore`.

## Logging

Debug logs are written to `/tmp/critic.log`. Control log level with the environment variable:

```bash
CRITIC_LOG_LEVEL=DEBUG critic
```
