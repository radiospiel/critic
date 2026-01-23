# Refactor Memoize to LRUCache

**Started**: 2026-01-23 03:22:30
**Ended**: 2026-01-23 03:30:00
**Strategy:** Refactoring
**Status:** Completed
**Complexity**: Simple
**Used Models:** Opus
**Token usage (Estimated):** 50k input, 12k output

## Summary
Refactor the memoize implementation to export LRUCache directly with a creator function, removing wrapper functions.

## Task
- Move lruCache into utils package as exported type
- Add creator function to LRUCache for automatic value creation
- Use LRUCache directly in consumers
- Remove Memoize1/Memoize2 functions

## Strategy
**Refactoring** - Code restructuring to simplify the API

## Progress
- [x] Export LRUCache in utils package
- [x] Add creator function parameter to NewLRUCache
- [x] Remove Set method (values created automatically via creator)
- [x] Remove default limit constant
- [x] Add precondition check for limit >= 1
- [x] Update fnmatch.go to use LRUCache directly
- [x] Remove Memoize1/Memoize2 functions
- [x] Update tests
- [x] Verify all tests pass

## Obstacles
(none)

## Outcome
Successfully refactored the memoize implementation:
- Renamed `lruCache` to `LRUCache` (exported)
- Renamed `newLRUCache` to `NewLRUCache` (exported)
- Added creator function parameter - called automatically when key not found
- Removed `Set` method - caching is now automatic via `Get`
- Removed `LRUCacheDefaultLimit` constant
- Added precondition: panics if limit < 1
- Removed `Memoize1` and `Memoize2` wrapper functions
- Updated `fnmatch.go` to use `LRUCache` directly with struct key and creator
- Updated tests to test `LRUCache` API directly
