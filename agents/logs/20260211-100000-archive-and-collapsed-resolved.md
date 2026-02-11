# Task: Archive conversations + collapsed resolved view

**Started:** 2026-02-11 10:00:00
**Ended:** 2026-02-11 10:30:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Medium
**Used Models:** Opus
**Token usage (Estimated):** 80k input, 20k output

## Objective
Add archived conversation status and collapsed resolved conversation view.

## Progress
- [x] Add StatusArchived/ConversationArchived to Go types (critic, messagedb)
- [x] Add CONVERSATION_STATUS_ARCHIVED to proto enum
- [x] Add ArchiveConversation and UnresolveConversation RPCs to proto
- [x] Regenerate Go and TypeScript protobuf code
- [x] Create archive_conversation.go and unresolve_conversation.go server handlers
- [x] Update criticStatusToApiStatus mapping
- [x] Add archiveConversation/unresolveConversation to frontend client.ts
- [x] Add collapsed resolved view to CommentDisplay.tsx
- [x] Add Archive/Unresolve buttons to CommentDisplay.tsx and FileList.tsx
- [x] Filter archived conversations in DiffView.tsx and FileList.tsx
- [x] Add showArchived toggle to App.tsx header
- [x] Add CSS styles for collapsed view, buttons, and toggle
- [x] Verify: go build, go test, tsc --noEmit all pass

## Obstacles
None encountered.

## Outcome
Full implementation of archive + collapsed resolved features. Backend supports archived status via new RPCs. Frontend shows resolved conversations in a collapsed single-line view with unresolve/archive buttons. Archived conversations are hidden by default with a header toggle to reveal them.

## Files Modified
### Backend
- `src/pkg/critic/messaging.go` — StatusArchived, ConversationArchived
- `src/api/proto/critic.proto` — ARCHIVED enum, Archive/Unresolve RPCs
- `src/messagedb/messagedb.go` — StatusArchived, MarkConversationAs case
- `src/messagedb/messaging.go` — convertToCriticStatus case
- `src/api/server/get_conversations.go` — status mapping
- `src/api/server/archive_conversation.go` — new handler
- `src/api/server/unresolve_conversation.go` — new handler
- Generated: `src/api/critic.pb.go`, `src/api/apiconnect/critic.connect.go`

### Frontend
- `src/webui/frontend/src/api/client.ts` — archived status, archive/unresolve functions
- `src/webui/frontend/src/components/CommentDisplay.tsx` — collapsed resolved view
- `src/webui/frontend/src/components/DiffView.tsx` — showArchived prop, archived filtering
- `src/webui/frontend/src/components/FileList.tsx` — showArchived prop, archive/unresolve buttons
- `src/webui/frontend/src/App.tsx` — showArchived state, archive toggle button
- `src/webui/frontend/src/index.css` — collapsed/archive styles
- Generated: `src/webui/frontend/src/gen/critic_pb.ts`, `src/webui/frontend/src/gen/critic_connect.ts`
