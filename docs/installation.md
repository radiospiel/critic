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

## MCP Server Configuration

Critic can run as an MCP (Model Context Protocol) server, enabling AI coding assistants like Claude Code to participate in human-in-the-loop code review workflows.

### Installing as an MCP Server

**For Claude Code:**

```bash
# Add critic as an MCP server (use absolute path to the binary)
claude mcp add critic -- /path/to/critic mcp

# Or if critic is in your PATH
claude mcp add critic -- critic mcp
```

**For other MCP clients**, add to your MCP configuration file:

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

### Verifying Installation

After installation, the AI assistant will have access to these tools:

| Tool | Description |
|------|-------------|
| `get_critic_conversations` | List conversation UUIDs (filter by `unresolved`/`resolved`) |
| `get_full_critic_conversation` | Get full conversation thread by UUID |
| `reply_to_critic_conversation` | Add a reply to a conversation |

## Configuring AI Agents for HITL Workflow

To enable human-in-the-loop code review, configure your AI coding agent to check for and respond to critic feedback. Add the following to your project's `CLAUDE.md` or `AGENTS.md`:

### Example CLAUDE.md Configuration

```markdown
### Human-in-the-Loop Code Review

**The `/critic` command:** When the user types `/critic`, check for and respond to reviewer feedback:

1. Call `get_critic_conversations(status: "unresolved")` to check for pending feedback
2. For each conversation, call `get_full_critic_conversation(uuid)` to read the feedback
3. Address the feedback in your code changes
4. Call `reply_to_critic_conversation(uuid, message)` to acknowledge or discuss
5. Wait for reviewer approval before proceeding

**Command variations:**
- `/critic` - Check for feedback and respond to all unresolved conversations
- `/critic wait` - Notify the user you're waiting, then check for feedback when they respond
- `/critic status` - Just report the count of unresolved conversations

If the critic MCP server is available:
- Proactively check for feedback after completing implementation tasks
- Always respond to reviewer comments before marking work as complete
- Request explicit approval for significant architectural decisions
```

### Workflow Example

1. **Developer starts AI coding session** with critic MCP server configured
2. **AI implements feature**, developer reviews in `critic webui`
3. **Human reviewer** adds inline comments on the diff
4. **Developer types `/critic`** to trigger feedback check
5. **AI reads feedback**, addresses it, and replies to acknowledge
6. **Human approves** or continues the conversation
7. **AI commits** once approved

### Interactive Review Session

For a more interactive workflow, the human can:

1. Run `critic webui` in a terminal
2. Tell the AI: `/critic wait`
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
