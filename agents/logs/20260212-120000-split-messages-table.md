# Split messages table into conversations + messages

| Field       | Value                        |
|-------------|------------------------------|
| Strategy    | Refactoring                  |
| Complexity  | Medium                       |
| Started     | 2026-02-12 12:00             |
| Ended       |                              |
| Outcome     |                              |

## Goal
Separate conversation-level data (status, file_path, lineno, etc.) from message-level data (author, message text, read_status) into two tables.

## Steps
1. [x] Read and understand current codebase
2. [ ] Update schema.go — v6 schema + migration
3. [ ] Update messagedb.go — new structs + DB methods
4. [ ] Update messaging.go — adapt interface implementation
5. [ ] Update messagedb_test.go — adapt tests
6. [ ] Update cli/test.go — rename method call
7. [ ] Run tests and fix issues

## Obstacles
(none yet)
