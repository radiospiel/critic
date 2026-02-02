# Critic Documentation

Critic is a mcp and web server which offers a git diff viewer and integrates code review comments. It enables human-in-the-loop code review workflows with AI assistants through MCP (Model Context Protocol) integration. It allows humans to review code changes made by the agent, and to ask for adjustments, without going through a github. This allows for a faster turn-around between changes and review, allowing the agent to actively employ a human-in-the-loop review process.

**Note that critic is designed as a single-user experience.** A typical scenario runs the critic mcp and http servers on the same machine that also runs the coding agent, and only listens on localhost. It is important to not deploy critic in an unsecured environment, since the web client has access to the entire tree of source files.

Remote installations on, for example, Claude Code for Web should be possible using ngrok or a similar reverse proxy with websocket support; this, however, has not been tested yet.
 
## Documentation

- [Installation](/docs/installation.md) - How to build and install Critic
- [Usage](/docs/usage.md) - Command-line options and keyboard shortcuts
- [Design](/docs/design.md) - System architecture and communication patterns
- [Hacking](/docs/hacking.md) - Testing and development guide
- [Plans](/docs/plans.md) - Roadmap and planned features

## Quick Start

```bash
# Clone and build
git clone https://github.com/radiospiel/critic.git
cd critic
make build

# Register critic as an MCP server
# (use absolute path to the binary)
claude mcp add critic -- /path/to/critic mcp

# Or if critic is in your PATH
claude mcp add critic -- critic mcp

# Start web UI
./critic webui --port=8080
```

Now you can open critic on http://localhost:8080.

Note that this quick start guide does not instruct the coding agent to use critic; an example prompt on how to use that is in [Installation](/docs/installation.md). This project also uses the critic mcp server, and has this section in its [CLAUDE.md](/CLAUDE.md) file:

```
## Ask for human reviewer approval

If the "critic" MCP server is available, but not on claude code for web:

- Before committing any significant code changes, call the get_review_feedback tool. Wait for reviewer approval before proceeding. Address any feedback in subsequent iterations.
```
