# Task: Branch Selector UX Improvements

**Started:** 2026-03-03 12:00:00
**Ended:** 2026-03-03 12:05:00
**Strategy:** Feature
**Status:** Completed
**Complexity:** Simple
**Used Models:** Opus
**Token usage (Estimated):** 30k input, 10k output

## Objective
Replace implicit click logic in branch selector with explicit Start/End hover buttons and add a dropdown chevron.

## Progress
- [x] Add dropdown chevron to trigger button
- [x] Replace row click with hover-reveal Start/End buttons
- [x] Remove hint text and handleNodeClick function
- [x] Add CSS for new elements
- [x] Remove unused .diff-graph-hint CSS
- [x] Verify build passes

## Obstacles
None.

## Outcome
All changes implemented. Build passes. Two files modified:
- `DiffBaseSelector.tsx`: Removed `handleNodeClick`, added chevron, replaced row onClick with Start/End buttons with disable logic
- `index.css`: Added `.diff-base-chevron`, `.diff-graph-actions` styles, removed `.diff-graph-hint`, changed row cursor from pointer to default
