# Task: Implement diffview line selection

**Started:** 2026-01-30 00:29:41
**Ended:** 2026-01-30 00:45:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Medium
**Used Models:** Opus
**Token usage (Estimated):** 100k input, 40k output

## Objective
Implement selected lines in the diffview with the following requirements:
- There is always a selected line in the diff view
- Up/down keys move the selected line up and down
- Shift+up/down jumps in steps of 10
- Switch to previous/next file at boundaries

## Progress
- [x] Explored codebase to understand diffview structure
- [x] Reviewed existing TUI selection implementation
- [x] Verified TUI selection is already fully implemented
- [x] Added comprehensive unit tests for TUI navigation behavior
- [x] Implemented web UI line selection in DiffView.tsx
- [x] Added CSS styles for selected line highlighting
- [x] Updated App.tsx to support file navigation from DiffView
- [x] Build passes successfully

## Obstacles
None

## Outcome

### TUI (Already Implemented)
The diffview line selection feature was already fully implemented in the Go TUI codebase:

1. **Always selected line**: `cursorLine` is always set to a navigable line (initialized to first navigable line on file load)
2. **Up/down navigation**: `moveCursorUp()` and `moveCursorDown()` functions
3. **Shift+up/down jump by 10**: `moveCursorUpN()` and `moveCursorDownN()` functions with `config.ShiftArrowJumpSize = 10`
4. **File switching at boundaries**: When at top/bottom and cannot move, triggers `session.SelectPrevFile()` or `session.SelectNextFile()`
5. **Visual selection**: Selection is highlighted via `buf.InvertRow()`

Added test file `src/tui/diffview_navigation_test.go` with 11 tests covering all navigation behavior.

### Web UI (Newly Implemented)
Implemented line selection in the React web UI:

1. **DiffView.tsx**: Added selection state, keyboard handlers (up/down/j/k, shift+up/down, g/G), scroll-to-view
2. **App.tsx**: Added file list tracking and navigation callbacks (`handleNavigatePrevFile`, `handleNavigateNextFile`)
3. **index.css**: Added `.diff-line-selected` styles with outline and brightness filter for visibility

Key features:
- `selectedLineIndex` state tracks current selection
- Keyboard handlers for navigation (arrow keys, j/k, shift modifier for 10-line jump)
- File boundary detection triggers prev/next file navigation
- Visual highlight with outline and brightness adjustment
- Works in both unified and split view modes

## Insights
The TUI already had a robust implementation that served as a good reference for the web UI implementation. The web UI required different approaches for some aspects (e.g., refs for scroll-into-view, different styling approach).
