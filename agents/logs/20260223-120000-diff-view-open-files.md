# Open Changed Files in Diff View

| Field       | Value                                |
|-------------|--------------------------------------|
| Strategy    | Feature                              |
| Complexity  | Medium                               |
| Started     | 2026-02-23 12:00                     |
| Ended       | 2026-02-23 12:05                     |
| Outcome     | Implemented, compiles clean           |

## Goal
When clicking a file in the Critic sidebar, open it in VS Code's diff editor showing base vs working copy, instead of plain editor.

## Steps
- [x] Add `getDiffBases()` to `criticClient.ts`
- [x] Create `baseFileProvider.ts` — virtual document provider for `critic-base` scheme
- [x] Update `extension.ts` — register provider, update `openFile` command
- [x] Update `fileListProvider.ts` — pass diff info in command args

## Obstacles
(none yet)
