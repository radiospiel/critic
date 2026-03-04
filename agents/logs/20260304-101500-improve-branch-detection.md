# Task: Improve branch detection (#128)

**Started:** 2026-03-04 10:15:00
**Ended:** 2026-03-04 10:35:00
**Strategy:** Refactoring
**Status:** Completed
**Complexity:** Medium
**Used Models:** Opus
**Token usage (Estimated):** 80k input, 20k output

## Objective
Simplify branch detection by moving from multiple diff bases to a single one,
using git log --decorate-refs for branch discovery, and removing custom sorting.

## Progress
- [x] Change DiffBases (plural list) to DiffBase (singular string) in config
- [x] Auto-detect master/main when no diff base configured, fail if neither exists
- [x] Replace LocalBranchesOnPath to use git log --decorate-refs with lo filtering
- [x] Remove SortRefsByGraphOrder and graphDistance (let git dictate order)
- [x] Remove SortByGraphOrder from GitOps interface
- [x] Update project.critic to use singular diffbase field
- [x] Update tests for new API
- [x] All tests pass

## Obstacles
None.

## Outcome
Branch detection simplified: single diff base in config, auto-detection of
master/main, git log-based discovery with lo filtering, no custom sorting.
