# Refactor Memoize to LRUCache

**Started**: 2026-01-23 03:22:30
**Ended**: 2026-01-23 03:35:00
**Strategy:** Refactoring
**Status:** Completed
**Complexity**: Medium
**Used Models:** Opus
**Token usage (Estimated):** 80k input, 20k output

## Summary
Refactor the memoize implementation to export LRUCache directly with a creator function that supports error handling.

## Task
- Move lruCache into utils package as exported type
- Add creator function to LRUCache for automatic value creation
- Support error handling in creator and Get
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
- [x] Rename order to usageOrder
- [x] Add error handling: creator returns (V, error), Get returns (V, error)
- [x] Don't cache error results
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
- Renamed `order` to `usageOrder` for clarity
- Added creator function `func(K) (V, error)` - called automatically when key not found
- Get now returns `(V, error)` - propagates creator errors
- Error results are not cached (retry on next Get)
- Removed `Set` method - caching is now automatic via `Get`
- Removed `LRUCacheDefaultLimit` constant
- Added precondition: panics if limit < 1
- Removed `Memoize1` and `Memoize2` wrapper functions
- Updated `fnmatch.go` to use `LRUCache` directly with struct key and creator
- Updated tests to test `LRUCache` API directly including error handling
