# Claude Code Session Integration

| Field       | Value                                    |
| ----------- | ---------------------------------------- |
| Started     | 2026-03-01 21:28                         |
| Ended       |                                          |
| Complexity  | Medium                                   |
| Outcome     |                                          |
| Strategy    | Feature (TDD)                            |

## Goal
Integrate critic UI with Claude Code sessions so reviewer can trigger prompts directly from the browser.

## Steps
- [x] DB: Add GetSetting/SetSetting to messagedb
- [x] CLI: Add set-setting/get-setting commands
- [x] Proto: Add SetClaudeSession, InjectPrompt RPCs + extend GetConfigResponse
- [x] Server: Implement new RPCs
- [x] MCP: Add /critic:activate prompt
- [x] Hook: capture-session-id.sh + settings.json
- [x] Frontend: Add injectPrompt client wrapper + Ask Claude buttons
- [x] Regenerate proto, verify builds
- [ ] Reviewer approval + commit

## Obstacles
(none)
