# Task: Add Memoize1 and Memoize2 utility functions

**Started:** 2026-01-21 14:33:04
**Ended:** 2026-01-21 14:38:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Simple
**Used Models:** Opus

## Objective
Add `utils.Memoize1` and `utils.Memoize2` functions that memoize function calls with one/two arguments, using generics and LRU caching with a 256 entry limit.

## Progress
- [x] Explore existing utils package structure
- [x] Write tests for Memoize1 function
- [x] Implement Memoize1 function
- [x] Write tests for Memoize2 and LRU behavior
- [x] Implement Memoize2 with LRU caching
- [x] Update Memoize1 to use LRU caching
- [x] Run tests to verify
- [x] Commit and push changes

## Obstacles
None.

## Outcome
Successfully added both memoization functions:
- `Memoize1[A comparable, R any](fn func(A) R) func(A) R`
- `Memoize2[A, B comparable, R any](fn func(A, B) R) func(A, B) R`

Both use slice-based LRU caching with 256 entry limit:
- Slice maintains access order (most recently used at end)
- Map provides O(1) lookups
- On cache hit, entry moves to end of slice
- On cache miss at capacity, oldest entry (index 0) is evicted

## Learnings
None - straightforward implementation following existing patterns.
