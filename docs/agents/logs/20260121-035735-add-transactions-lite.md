# Task: Add transactions lite to observable

**Started:** 2026-01-21 03:57:35
**Ended:** 2026-01-21 06:30:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Medium
**Used Models:** Opus

## Objective
Implement a "transactions lite" feature for the observable package with:
1. Callback-based transaction API: `obs.Transaction(func(tx *Txn) { ... })`
2. `Txn.SetValueAtKey()` records changes in a slice (doesn't apply immediately)
3. Auto-commit on callback return, with `Txn.Abort()` to cancel
4. Parent key changes override child key changes (e.g., setting "a" after "a.1.b" discards "a.1.b")
5. Simplified notification semantics (no value tree walking)

## Progress
- [x] Explore existing observable implementation
- [x] Initial implementation with channel-based buffering
- [x] Fix unbounded buffer issue (replace channel with mutex-protected map)
- [x] Redesign with proper `Begin()` / `Commit()` API
- [x] Change to callback-based API with auto-commit
- [x] Merge TransactionalObservable into Observable (all observables are transactional)
- [x] Implement change deduplication (parent overrides children)
- [x] Simplify notification semantics (no value tree walking)
- [x] Change ChangeCallback from `func(obs *Observable, key string)` to `func(key string)`
- [x] Move `matchPattern` to `must.Fnmatch`
- [x] Remove unused `obs` field from `Txn` struct
- [x] Run tests and verify (all 68 tests pass)

## Obstacles
- **Issue:** Initial channel-based design had 1000-item buffer limit causing potential deadlock
  **Resolution:** Replaced with mutex-protected map for unbounded buffering

- **Issue:** User requested callback-based API instead of Begin()/Commit()
  **Resolution:** Changed to `obs.Transaction(func(tx *Txn) { ... })` with auto-commit

- **Issue:** Complex value tree walking for notifications was over-engineered
  **Resolution:** Simplified to key-hierarchy based notifications only

## Outcome
Successfully implemented transactions on Observable with:
- `obs.Transaction(func(tx *Txn) { ... })` - callback-based with auto-commit
- `tx.SetValueAtKey(key, value)` / `tx.DeleteValueAtKey(key)` - record changes
- `tx.Abort()` - cancel transaction, discard all changes
- `keyOverrides(parent, child)` / `keyAffectsPattern(key, pattern)` for deduplication and notification
- Simplified notification: setting parent notifies all child subscriptions

Key design decisions:
- No goroutines needed - simple synchronous model
- Callback-based API ensures transactions are always completed
- Txn struct doesn't need Observable reference (closure captures it)
- No value tree walking - notifications based on key hierarchy only
- Pattern validation moved to `must.Fnmatch` for fail-fast behavior

## Learnings
- Callback-based transaction APIs (like `db.Transaction(func(tx) { ... })`) are cleaner than explicit Begin/Commit
- Simpler notification semantics (key hierarchy only) are easier to reason about
- `path.Match` uses "/" as separator, so "*" matches "a.b" (dots are not separators)
- Consolidating related functionality (keyOverrides ≈ keyAffectsPattern) reduces duplication
