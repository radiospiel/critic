# Preserve DiffView state across live reloads

- **Strategy**: Bug Fix
- **Complexity**: Simple
- **Started**: 2026-03-01 11:55
- **Ended**: 2026-03-01 11:58
- **Outcome**: Completed

## Problem
When the backend sends a WebSocket `reload` message, the DiffView component would:
1. Scroll to the restored line position (forced `scrollIntoView`)
2. Reset view mode to `'diff'`
3. Close the comment editor

## Solution
Added a `prevPathRef` to track the previously displayed file path. Used it to distinguish "new file selected" from "same file reloaded with updated data":
- View mode reset: only when file path changes
- Editor close: only when file path changes
- Scroll jump: only on explicit navigation, not on data reload

Extracted `findBestLineIndex` helper to avoid duplicating line-search logic.

## Files Changed
- `src/webui/frontend/src/components/DiffView.tsx`

## Obstacles
None.
