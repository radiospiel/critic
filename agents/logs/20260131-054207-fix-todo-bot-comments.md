# Task: Address TODO(bot) comments in codebase

**Started:** 2026-01-31 05:42:07
**Ended:** 2026-01-31 08:56:33
**Strategy:** Mixed (Feature, Refactoring, Bug Fix)
**Status:** Completed (7 of 9 items - remaining 2 require TUI/CLI design work)
**Complexity:** Medium
**Used Models:** Opus

## Objective
Find and address all TODO(bot) comments in the codebase.

## TODO(bot) Items Found
1. `critic-dev:9` - Feature: Add --cpuprofile flag for CPU profiling
2. `simple-go/utils/utils.go:114` - Bug Fix: Race condition on initial map read without lock
3. `src/api/server/create_comment.go:35` - Refactoring: Replace validation with JSON schema
4. `src/api/server/get_comments_test.go:3` - Feature: Build tests for GetComments GRPC action
5. `src/api/server/session.go:39` - Feature: Allow users to switch diffBases from UI (B hotkey)
6. `src/api/server/session.go:162` - Feature: Add filtering by type using rg --type-list
7. `src/api/server/server.go:65` - Refactoring: Pass config.DiffBases into NewSession
8. `src/api/server/get_comments.go:13` - Refactoring: Reimplement other GRPC methods following pattern
9. `src/api/server/get_comments.go:18` - Feature: Adjust webui to fetch comments from grpc call

## Progress
- [x] Identified all TODO(bot) items
- [x] Fix LRU cache race condition (utils.go) - Changed Mutex to RWMutex, protected initial map read
- [x] Write tests for GetComments (get_comments_test.go) - Comprehensive test coverage
- [x] Pass DiffBases into NewSession (server.go/session.go) - Refactored signature
- [x] Refactor CreateComment to follow GetComments pattern (create_comment.go) - Uses depanic wrapper
- [x] Add JSON schema validation (create_comment.go) - Added schema in schema.go
- [x] Add --cpuprofile flag (api.go) - Added to api command
- [x] Refactor all GRPC handlers to use depanic pattern (get_diff.go, get_diff_summary.go, get_file.go, get_last_change.go)
- [x] Adjust webui to fetch comments from grpc - Updated client.ts, regenerated TypeScript types
- [ ] Add type filtering (session.go) - Requires CLI changes and more design
- [ ] Add UI diff base switching (session.go) - Requires TUI work

## Obstacles
None significant. The remaining items require TUI work or CLI design decisions.

## Outcome
Completed 7 out of 9 TODO(bot) items:
- Fixed LRU cache race condition
- Added comprehensive GetComments tests
- Refactored NewSession to accept DiffBases
- Refactored CreateComment to follow GetComments pattern
- Added JSON schema validation for CreateComment
- Added --cpuprofile flag for CPU profiling
- Refactored all GRPC handlers to use depanic wrapper pattern
- Updated webui to fetch comments via GRPC instead of REST

Remaining 2 items need additional work:
- Type filtering requires CLI design and implementation
- UI diff base switching requires TUI work

## Insights
- The codebase has a well-established pattern for GRPC handlers using `depanic` wrappers
- JSON schema validation is centralized in schema.go
- Frontend uses Connect-RPC with generated TypeScript types from proto files
- Proto regeneration: `cd src/webui/frontend && npx buf generate ../../api/proto`
