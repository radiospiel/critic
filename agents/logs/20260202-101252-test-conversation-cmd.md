# Add CLI Test Conversation Command

**Started:** 2026-02-02 10:12
**Ended:** 2026-02-02 10:15
**Complexity:** Simple
**Outcome:** Complete

## Task
Add CLI subcommand `critic test conversation file:line "text"` which:
1. Creates a conversation on the given file/line
2. If conversation already exists at that file/line, reply to it instead

## Strategy
Feature (TDD) - New functionality

## Implementation Plan
1. Create new file `src/cli/test.go` with:
   - `newTestCmd()` - parent "test" command
   - `newTestConversationCmd()` - the conversation subcommand
2. Register test command in `parser.go`
3. Parse `file:line` format
4. Use `GetMessagesByFile` to check if conversation exists
5. Use `CreateConversation` or `ReplyToConversation` accordingly

## Progress
- [x] Explore codebase structure
- [x] Implement test command
- [x] Build and test
- [x] Commit

## Obstacles
(None)
