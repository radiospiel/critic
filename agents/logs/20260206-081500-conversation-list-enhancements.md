# Conversation List Enhancements

| Field       | Value                                      |
|-------------|--------------------------------------------|
| Strategy    | Feature                                    |
| Complexity  | Medium                                     |
| Started     | 2026-02-06 08:15                           |
| Ended       | 2026-02-06 08:17                           |
| Outcome     | Complete, awaiting review                   |

## Task

Enhance conversation list in FileList component:
1. Add "Resolve" button for unresolved conversations
2. Render unresolved conversations in blue (not orange/yellow)
3. Don't show line numbers
4. Correctly pluralize "message" (not "msg")
5. Add timestamp to latest message
6. Don't indent root conversation, indent file conversations to match file name indentation
7. Consistently render "unresolved" as "open"

## Progress

- [ ] Read and understand current implementation
- [ ] Implement all changes in FileList.tsx
- [ ] Update CSS styles (blue for unresolved)
- [ ] Test and verify

## Obstacles
