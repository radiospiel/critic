# Order Branches by Age

- **Started**: 2026-03-01 12:00:00
- **Ended**: 2026-03-01 12:30:00
- **Complexity**: Medium
- **Strategy**: Feature (TDD)

## Task

Extend the branch selection dropdown in the UI:
1. Order branches by graph distance from HEAD (oldest first in data model)
2. Backend discovers all local branches on the ancestry path from oldest to HEAD
3. UI displays in reverse order: newest first (working dir, HEAD, then older)

## Progress

- [x] Add git utility: graph distance from HEAD (replaces timestamp-based)
- [x] Add git utility: discover local branches on ancestry path
- [x] Add git utility: sort branches by graph order
- [x] Write tests for branch discovery
- [x] Update cli/parser.go to use branch discovery
- [x] Update DiffBaseSelector.tsx for reverse display order
- [x] Run tests and verify

## Obstacles

- Initial implementation used commit timestamps for ordering; user feedback corrected this to use graph walking (topological distance from HEAD), which is more reliable than timestamps (which can be wrong after rebases/amends).

## Outcome

Completed. Three files changed in backend (mergebase.go, parser.go), one in frontend (DiffBaseSelector.tsx), plus new test file (mergebase_test.go).
