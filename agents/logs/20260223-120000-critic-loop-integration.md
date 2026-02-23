# Task: Critic loop integration via Stop hook

**Started:** 2026-02-23 12:00:00
**Ended:** 2026-02-23 12:15:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Medium
**Used Models:** Opus
**Token usage (Estimated):** 80k input, 20k output

## Objective
Add a Stop hook to Claude Code that checks for unresolved critic conversations via MCP tools before allowing Claude to go idle. This ensures the human reviewer's feedback is addressed before the session ends.

## Progress
- [x] Explored codebase and understood MCP server, hooks, and critic loop
- [x] Read existing hook configuration and files
- [x] Create critic-loop-on-idle.sh guard script (re-entry prevention)
- [x] Create critic-loop-prompt.md for Stop hook (MCP-based check)
- [x] Update .claude/settings.json with Stop hook
- [x] Commit and push

## Obstacles
- **Issue:** Stop hooks can cause infinite loops since "unresolved" conversations remain unresolved until the human marks them resolved in the Critic UI.
  **Resolution:** Used an alternating guard file pattern — the guard script creates a file on first stop, then removes it on second stop. Combined with the prompt checking if the last message is from AI (already addressed), this prevents re-entry while still catching new feedback.

## Outcome
Three files added/modified:
- `agents/hooks/critic-loop-on-idle.sh` — Guard script preventing infinite re-entry
- `agents/hooks/critic-loop-prompt.md` — Prompt instructing Claude to check MCP tools for unresolved feedback
- `.claude/settings.json` — Stop hook configuration added alongside existing PreToolUse hook

## Insights
- The prompt-level check (step 4: "if last message is from AI, you may finish") is the semantic guard. The guard file is the mechanical guard. Both are needed: the semantic check handles the case where conversations are unresolved but already addressed; the guard file handles the case where Claude's response to the prompt itself triggers another stop.
- Future improvement: adding a `--unread-by-ai` filter to `get_critic_conversations` would make the check naturally idempotent and eliminate the need for the guard file.
