# Task: Add transactions lite to observable

**Started:** 2026-01-21 03:57:35
**Ended:** 2026-01-21 04:25:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Medium
**Used Models:** Opus

## Objective
Implement a "transactions lite" feature for the observable package with:
1. `Begin()` → `Transaction` function to start a transaction
2. `Transaction.SetValueAtKey()` records changes in a slice (doesn't apply immediately)
3. `Transaction.Commit()` deduplicates changes and applies them atomically
4. Parent key changes override child key changes (e.g., setting "a" after "a.1.b" discards "a.1.b")

## Progress
- [x] Explore existing observable implementation
- [x] Initial implementation with channel-based buffering
- [x] Fix unbounded buffer issue (replace channel with mutex-protected map)
- [x] Redesign with proper `Begin()` / `Commit()` API
- [x] Implement change deduplication (parent overrides children)
- [x] Add `setValuesAtKeys` internal method
- [x] Update tests for new API
- [x] Run tests and verify (all 73 tests pass)

## Obstacles
- **Issue:** Initial channel-based design had 1000-item buffer limit causing potential deadlock
  **Resolution:** Replaced with mutex-protected map for unbounded buffering

- **Issue:** User requested proper transaction API with Begin()/Commit() instead of implicit buffering
  **Resolution:** Complete redesign with Transaction type that records changes in a slice

- **Issue:** `path.Match("*", "a.final")` returns true because "." is not a separator in path matching
  **Resolution:** Updated tests to use specific key patterns instead of relying on "*" wildcard behavior

## Outcome
Successfully implemented TransactionalObservable with:
- `Begin()` returns a `Transaction` object
- `Transaction.SetValueAtKey(key, value)` records changes in a slice
- `Transaction.Commit()` deduplicates and applies changes atomically
- `keyOverrides(parent, child)` determines if parent change overrides child
- Example: `["a.1.b", "a", "a.2", "c", "a.1", "a"]` → only `["c", "a"]` applied

Key design decisions:
- No goroutines needed - simple synchronous model
- Changes recorded as slice of {key, value} pairs
- Deduplication works backwards from end of slice
- Setting parent key overrides all child key changes
- Notifications happen once per unique changed key

## Learnings
- Channel-based designs need careful consideration of buffer limits
- `path.Match` uses "/" as separator, so "*" matches "a.b" (dots are not separators)
- Explicit transaction objects (Begin/Commit) are cleaner than implicit buffering
- Deduplication logic: iterate backwards, skip changes overridden by later parent changes
