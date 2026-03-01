# Move Category Assignment to Backend

| Field       | Value                                      |
|------------|---------------------------------------------|
| Started    | 2026-02-28 10:00                            |
| Ended      | 2026-03-01 10:30                            |
| Complexity | Medium                                      |
| Strategy   | Refactoring                                 |
| Outcome    | Success                                     |

## Goal
Move file category assignment from frontend to backend. Extend API protos so each file carries its category. Use fnmatch.go (extended with `**` support) for pattern matching. Remove frontend pattern matching code.

## Steps
- [x] Extend fnmatch.go with `**` (doublestar) support
- [x] Rewrite pathspec.go to use fnmatch internally (with negation support)
- [x] Add `category` field to proto `FileSummary`
- [x] Regenerate protos (Go + TypeScript)
- [x] Populate category in backend's convertFileSummary
- [x] Update frontend to use category from API
- [x] Remove frontend glob.ts and categorizeFile function
- [x] Write unit tests covering project.critic cases
- [x] Run tests — all pass

## Obstacles
- protoc not installed — resolved by using buf CLI downloaded from GitHub
