# Extend GetConversations + add batch GetFullConversations

- **Strategy**: Feature (TDD)
- **Complexity**: Medium
- **Started**: 2026-02-12 12:00
- **Ended**:

## Goal
1. Add `paths` filter to `GetConversations` so root selection happens in one query
2. Add `GetFullConversations(uuids)` to batch-fetch messages for multiple conversations
3. Update all callers

## Progress
- [x] Update interface in `messaging.go`
- [x] Implement in `messagedb/messaging.go`
- [x] Update callers (api server, mcp, cli)
- [x] Update tests (messagedb + integration)
- [x] Verify build + tests — all pass
- [ ] Reviewer approval

## Obstacles
(none)

## Outcome
Pending review.
