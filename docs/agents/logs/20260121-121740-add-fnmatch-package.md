# Task: Add fnmatch package

**Started:** 2026-01-21 12:17:40
**Ended:** 2026-01-21 12:50:36
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Simple
**Used Models:** Opus

## Objective
Create a new `simple-go/fnmatch` package that provides shell-style pattern matching (fnmatch) functionality by converting patterns to regex and providing an LRU cache for compiled patterns. Then replace `must.Fnmatch` with the new package.

## Progress
- [x] Create task progress log
- [x] Write fnmatch_test.go with comprehensive tests
- [x] Implement fnmatch.go with Fnmatcher type and functions
- [x] Run tests and verify all pass
- [x] Commit and push initial implementation
- [x] Rebase to latest master
- [x] Add FnmatchPath for path-style matching (where * doesn't match dots)
- [x] Replace must.Fnmatch usage in observable with fnmatch.FnmatchPath
- [x] Remove must.Fnmatch from must package
- [x] Run all tests and push final changes

## Obstacles
- **Issue:** Initial test expectations for regex conversion didn't match the specified behavior (e.g., `*` matches any character including `/`)
  **Resolution:** Fixed test expectations to match the user's specification where `*` converts to `.*`

- **Issue:** must.Fnmatch had different semantics (used path.Match where `*` doesn't cross segment boundaries)
  **Resolution:** Added FnmatchPath variant that uses `[^.]*` instead of `.*` to preserve segment-aware matching

## Outcome
Successfully implemented the `simple-go/fnmatch` package with:
- `fnmatchToRegex()` function that converts fnmatch patterns to regex
- `Fnmatcher` struct with `Match()` method
- `MustCompile()` function to create compiled matchers
- `Fnmatch()` convenience function with LRU cache (256 capacity) - `*` matches everything
- `MustCompilePath()` for path-style pattern compilation
- `FnmatchPath()` convenience function with LRU cache - `*` matches single segment only (not dots)
- Comprehensive test coverage for all functions
- Removed `must.Fnmatch` and updated observable to use new package

## Learnings
- TDD approach helped catch test expectation mismatches early
- When replacing existing functionality, preserve semantics by understanding how it was used
- The fnmatch package now provides two modes: greedy (`Fnmatch`) and segment-aware (`FnmatchPath`)
