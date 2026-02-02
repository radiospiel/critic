# Task: Implement gRPC Conversation Endpoints

**Strategy**: Feature (TDD)
**Started**: 2026-01-31 10:00
**Ended**: 2026-01-31 12:30
**Complexity**: Medium
**Outcome**: Success

## Summary

Implement gRPC endpoints for conversation management:
1. Rename `CreateComment` → `CreateConversation`
2. Rename `GetComments` → `GetConversations`
3. Add new `GetConversationsSummary` endpoint
4. Update frontend to show conversation icons in file list

## Requirements

- `CreateConversation`: Store a comment (as a conversation) - existing functionality renamed
- `GetConversations`: Load all conversations for a specific file - existing functionality renamed
- `GetConversationsSummary`: List number of conversations and status per filepath (only files with conversations)
- Frontend: Show icon next to filepath with conversations, animated if unresolved

## Progress

- [x] Explore codebase and understand existing structure
- [x] Update proto file with renamed endpoints and new GetConversationsSummary
- [x] Regenerate Go and TypeScript protobuf code
- [x] Rename backend handlers (CreateComment → CreateConversation, GetComments → GetConversations)
- [x] Implement GetConversationsSummary backend handler
- [x] Add GetAllFileConversationSummaries to messaging interface
- [x] Update frontend API client
- [x] Add conversation icon to FileList component
- [x] Add tests for new endpoint
- [ ] Test manually in fixtures repo

## Obstacles

(none)

## Changes Made

### Backend (Go)

1. **Proto file** (`src/api/proto/critic.proto`):
   - Renamed `CreateComment` → `CreateConversation`
   - Renamed `GetComments` → `GetConversations`
   - Added `GetConversationsSummary` RPC endpoint
   - Added `FileConversationSummary` message type

2. **Messaging interface** (`src/pkg/critic/messaging.go`):
   - Added `GetAllFileConversationSummaries()` method to interface
   - Extended `FileConversationSummary` with `TotalCount`, `UnresolvedCount`, `ResolvedCount` fields
   - Updated `DummyMessaging` implementation

3. **MessageDB** (`src/messagedb/messaging.go`):
   - Updated `getFileConversationSummary` to populate count fields
   - Added `GetAllFileConversationSummaries()` implementation

4. **API handlers** (`src/api/server/`):
   - Renamed `create_comment.go` → `create_conversation.go`
   - Renamed `get_comments.go` → `get_conversations.go`
   - Added `get_conversations_summary.go`
   - Updated tests accordingly

5. **Schema validation** (`src/api/schema.go`):
   - Renamed `/critic.v1.CriticService/CreateComment` → `/critic.v1.CriticService/CreateConversation`

### Frontend (TypeScript/React)

1. **API client** (`src/webui/frontend/src/api/client.ts`):
   - Renamed `getComments` → `getConversations` (with backward-compatible alias)
   - Added `getConversationsSummary()` function
   - Added `ConversationSummary` interface

2. **FileList component** (`src/webui/frontend/src/components/FileList.tsx`):
   - Fetches conversation summaries on mount
   - Displays conversation count badge next to files with conversations
   - Animates badge when file has unresolved conversations

3. **CommentEditor** (`src/webui/frontend/src/components/CommentEditor.tsx`):
   - Updated to use `createConversation` instead of `createComment`

4. **CSS** (`src/webui/frontend/src/index.css`):
   - Added `.conversation-icon` styles with pulse animation for unresolved

## Notes

- The messaging interface already has `getFileConversationSummary` which was extended
- Conversations are already rendered in DiffView, so that functionality was unchanged
- Legacy `getComments` alias maintained for backward compatibility
