# Refactor Diff Requests

**Strategy:** Refactoring
**Started:** 2026-01-28 12:00
**Ended:** 2026-01-28 12:45
**Complexity:** Medium

## Goal

Refactor the diff API to separate summary and detail requests:
1. Rename `GetDiffs` → `GetDiffSummary`, drop hunks from response
2. Add `git.GetDiffNamesBetween()` function using `git diff --name-status`
3. Add `GetDiff` request for single-file diff with hunks

## Progress

- [x] Update proto file with new messages (GetDiffSummary, GetDiff, DiffSummary, FileSummary)
- [x] Add `GetDiffNamesBetween` function in git package
- [x] Add `ParseDiffNameStatus` function in git package to parse --name-status output
- [x] Rename server handler to use GetDiffSummary behavior (hunks not included)
- [x] Create summary-only conversion functions
- [x] Add `filterDiffByPaths` helper function
- [x] Update `Session.SetRefs` to use `GetDiffNamesBetween` for efficiency
- [x] Add `Session.GetFileDiff` method for on-demand single-file diff loading
- [x] Update tests for new functions
- [ ] Regenerate protobuf files (blocked: protoc not available, proto file updated)
- [ ] Add new `GetDiff` RPC handler (blocked: requires proto regeneration)

## Obstacles

1. **protoc not available**: Cannot regenerate protobuf files. The proto file has been updated with the new schema, but generated Go/TypeScript files need to be regenerated when tooling becomes available.

## Notes

- The existing `GetDiffs` endpoint now returns summary only (no hunks) - this is a behavior change
- The proto file defines the new schema with `GetDiffSummary`, `GetDiff`, `DiffSummary`, `FileSummary`
- The server uses `GetDiffNamesBetween` to load only file metadata (more efficient)
- `GetFileDiff` method added to Session for on-demand loading of single-file diffs
- When protos are regenerated:
  - Rename `GetDiffs` handler to `GetDiffSummary`
  - Add `GetDiff` handler that uses `Session.GetFileDiff`
  - Update frontend to use new endpoint names

## Files Changed

- `src/api/proto/critic.proto` - New schema with GetDiffSummary, GetDiff, DiffSummary, FileSummary
- `src/git/diff.go` - Added GetDiffNamesBetween function
- `src/git/parser.go` - Added ParseDiffNameStatus function
- `src/git/parser_test.go` - Added tests for ParseDiffNameStatus
- `src/api/server/get_diff_summary.go` - Renamed from get_diffs.go, implements summary-only behavior
- `src/api/server/get_diff_summary_test.go` - Renamed from get_diffs_test.go, updated tests
- `src/api/server/session.go` - Added GetDiffSummary, GetFileDiff, filterDiffByPaths; updated SetRefs
- `src/api/server/session_test.go` - Added test for filterDiffByPaths, updated GetDiff → GetDiffSummary
