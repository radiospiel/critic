# Task: Add API integration tests using testsh

**Started:** 2026-02-22 22:10:00
**Ended:** 2026-02-22 22:25:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Simple
**Used Models:** Opus
**Token usage (Estimated):** 80k input, 10k output

## Objective
Add bash-based API integration tests in tests/integration/api/ using the testsh framework from github.com/radiospiel/testsh. Tests verify the critic HTTP server API endpoints end-to-end: creating a git repo, starting the server, posting comments, adding replies, and checking that GetLastChange timestamps update.

## Progress
- [x] Research codebase: proto definitions, server startup, Connect-RPC JSON format
- [x] Fetch and understand testsh framework
- [x] Create tests/integration/api/ directory with testsh.inc, test_api.sh, Makefile, .gitignore
- [x] Fix testsh.inc arithmetic to be compatible with set -euo pipefail
- [x] Run all 5 tests (13 assertions) — all pass
- [x] Wire API tests into parent integration Makefile

## Obstacles
- **Issue:** `((_passed++))` in testsh.inc causes exit code 1 when _passed is 0 under `set -e` (bash post-increment evaluates old value)
  **Resolution:** Changed to `_passed=$((_passed + 1))` which always succeeds

## Outcome
5 integration tests with 13 assertions covering the full conversation lifecycle through the HTTP API:
1. `test_get_last_change_returns_timestamp` — basic endpoint check
2. `test_create_conversation_and_verify_last_change` — comment creation updates mtime
3. `test_conversation_is_in_get_conversations` — comment content in response
4. `test_reply_and_verify_last_change` — reply updates mtime
5. `test_reply_is_in_conversation` — both comment and reply in response

## Insights
- Connect-RPC uses camelCase JSON field names (e.g., `mtimeMsecs`, `newFile`)
- The DB watcher polls every 1s, so tests need ~1.5s sleep after mutations to see mtime changes
