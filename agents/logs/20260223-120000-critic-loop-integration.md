# Task: Critic loop integration via Stop hook

**Started:** 2026-02-23 12:00:00
**Ended:** 2026-02-23 12:30:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Medium
**Used Models:** Opus
**Token usage (Estimated):** 120k input, 30k output

## Objective
Add a Stop hook to Claude Code that checks for unresolved critic conversations via MCP tools before allowing Claude to go idle. Extend the MCP tool and database layer with an "actionable" filter so the check is server-side rather than prompt-based.

## Progress
- [x] Explored codebase and understood MCP server, hooks, and critic loop
- [x] Read existing hook configuration and files
- [x] Create critic-loop-on-idle.sh guard script (re-entry prevention)
- [x] Create critic-loop-prompt.md for Stop hook
- [x] Update .claude/settings.json with Stop hook
- [x] Write failing test for "actionable" filter
- [x] Implement "actionable" filter in messagedb GetConversations
- [x] Update MCP tool description to document "actionable" status
- [x] Simplify Stop hook prompt to use "actionable" filter
- [x] Run all tests — all pass
- [x] Commit and push

## Obstacles
- **Issue:** Stop hooks can cause infinite loops since "unresolved" conversations remain unresolved until the human marks them resolved in the Critic UI.
  **Resolution:** Two-layer approach: (1) Guard script alternates between firing and skipping to prevent mechanical re-entry. (2) New "actionable" filter at the database level only returns conversations where the last message is from a human — once AI replies, the conversation drops out of the result set automatically.

## Outcome
Changes across 5 files:
- `agents/hooks/critic-loop-on-idle.sh` — Guard script preventing infinite re-entry
- `agents/hooks/critic-loop-prompt.md` — Prompt using `get_critic_conversations(status: "actionable")`
- `.claude/settings.json` — Stop hook configuration
- `src/messagedb/messaging.go` — "actionable" filter: unresolved + last message from human
- `src/mcp/server.go` — Updated tool description documenting "actionable" status
- `src/messagedb/messagedb_test.go` — Test covering all actionable filter scenarios

## Insights
- The "actionable" filter makes the critic loop naturally idempotent: once Claude replies, the conversation is no longer actionable. This is more robust than tracking read status or using prompt-level logic.
- The guard script is still needed as a mechanical safeguard against the edge case where `get_critic_conversations(status: "actionable")` returns results but Claude fails to reply (e.g., tool error). Without the guard, it would re-fire indefinitely.
