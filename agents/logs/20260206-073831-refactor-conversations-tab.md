# Task: Refactor Conversations tab to list all conversations grouped by file

**Started:** 2026-02-06 07:38:31
**Ended:** in progress
**Strategy:** Refactoring
**Status:** In Progress
**Complexity:** Medium
**Used Models:** Opus
**Token usage (Estimated):** 150k input, 30k output

## Objective
Refactor the Conversations tab in the sidebar to:
1. List ALL conversations grouped by file (not just file-level counts)
2. Show the root conversation at the top
3. Render unresolved conversations in yellow
4. Show first 150 chars of last message per conversation in smaller italic font

## Progress
- [x] Explored codebase structure
- [x] Read FileList.tsx, App.tsx, client.ts, index.css
- [x] Implement conversations view in FileList
- [x] Add root conversation display
- [x] Add CSS styles
- [x] TypeScript type check passes
- [x] Vite build succeeds
- [x] Go build succeeds
- [ ] Human review

## Obstacles
None.

## Changes Made
1. **FileList.tsx**: Added `getConversations` import, `CommentConversation` type, `rootConversation` prop, `fileConversations` state, effect to fetch full conversations, `truncateText` helper, and new rendering branch for conversations view with grouped-by-file display
2. **App.tsx**: Passed `rootConversation` prop to `FileList`
3. **index.css**: Added styles for `.conversation-group`, `.conversation-group-header`, `.conversation-entry`, `.conversation-entry-info`, `.conversation-entry-line`, `.conversation-entry-status`, `.conversation-entry-messages`, `.conversation-entry-preview`

## Outcome
Pending human review.

## Insights
- The `getConversations` API already exists per-file; fetching for all files with conversations in parallel works well for the grouped view.
