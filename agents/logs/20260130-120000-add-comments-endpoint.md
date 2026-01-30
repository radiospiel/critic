# Task: Add comments storage and retrieval endpoints

**Started:** 2026-01-30 12:00:00
**Ended:** 2026-01-30 12:45:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Medium
**Used Models:** Opus

## Objective
Implement endpoints to store and retrieve comments:
1. Modify CreateComment endpoint to persist comments using `messaging.CreateConversation`
2. Add new GetComments RPC method to load messages for a file using `GetConversationsForFile`
3. Update frontend to display the comments

## Progress
- [x] Implement CreateComment endpoint to persist comments
- [x] Add GetComments RPC method to proto file
- [x] Implement GetComments HTTP handler (protobuf code couldn't be regenerated due to network issues)
- [x] Update frontend to show comments
- [x] Test the implementation

## Obstacles
- **Issue:** protoc/buf not available due to network issues preventing package installation
  **Resolution:** Created a temporary HTTP/JSON endpoint (`/api/comments`) that works alongside the existing gRPC system. The proto file has been updated, so when protoc becomes available, the code can be regenerated and the endpoint can migrate to gRPC.

## Outcome
Successfully implemented comment storage and retrieval:
- CreateComment endpoint now persists comments to the database using `messaging.CreateConversation`
- GetComments HTTP endpoint returns all conversations for a file
- Frontend fetches and displays comments inline in the diff view
- Comments are shown after the line they're attached to
- After saving a comment, the view refreshes to show the new comment

## Insights
- When adding new gRPC methods without protoc available, an HTTP/JSON workaround can be implemented alongside the existing gRPC endpoints
- The proto file should still be updated for future regeneration
