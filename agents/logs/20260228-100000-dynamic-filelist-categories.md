# Dynamic file list categories from project config

| Field       | Value                                    |
|------------|------------------------------------------|
| Strategy   | Feature (TDD)                            |
| Complexity | Medium                                   |
| Started    | 2026-02-28 10:00                         |
| Ended      | 2026-02-28 10:03                         |
| Outcome    | Success — build passes, tabs dynamic     |

## Goal
Replace hardcoded file list filter tabs (Conversations/Files/Tests/Hidden) with dynamic tabs driven by `categories` array in `project.critic`.

## Progress
- [x] Update `FilterType` to `string` in FileList.tsx
- [x] Replace hardcoded file categorization with dynamic `Map<string, FileSummary[]>`
- [x] Render filter buttons dynamically from categories
- [x] Update `displayedFiles` logic
- [x] Simplify empty state messages
- [x] Update keyboard shortcuts in App.tsx
- [x] Watch project.critic file for changes (backend + frontend)
- [x] Build and verify (frontend + Go + tests)

## Obstacles
(none)
