# Task: Fix unchanged diff lines having wrong background color

**Started:** 2026-01-21 03:28:16
**Ended:** 2026-01-21 03:45:00
**Strategy:** Bug Fix
**Status:** Completed
**Complexity:** Simple
**Used Models:** Opus

## Objective
Fix a visual bug where unchanged lines (context lines) in the diff view have a light green background instead of a dark background with syntax highlighting.

## Progress
- [x] Explore codebase to understand diff rendering architecture
- [x] Identify root cause - found bug in ANSI color parsing
- [x] Implement fix in teapot/ansi.go
- [x] Add regression tests in teapot/ansi_test.go
- [x] Verify all tests pass
- [x] Commit and push

## Obstacles
- **Issue:** Pre-existing failing test `TestLineDisplacement` in integration tests
  **Resolution:** Confirmed this test was already failing before the fix was applied (unrelated to the ANSI parsing changes)

## Outcome
Fixed the ANSI color parsing bug in `teapot/ansi.go`. The issue was that basic ANSI color codes (30-47 for foreground/background, 90-107 for bright colors) were being passed directly to lipgloss, which interpreted them as 256-color palette indices rather than the standard ANSI colors they represent.

For example, ANSI code 40 means "black background" in the standard ANSI scheme, but lipgloss was interpreting it as palette color 40, which is green (#00d700).

The fix converts ANSI codes to their correct palette indices:
- Foreground 30-37 → palette 0-7 (subtract 30)
- Background 40-47 → palette 0-7 (subtract 40)
- Bright foreground 90-97 → palette 8-15 (subtract 82)
- Bright background 100-107 → palette 8-15 (subtract 92)

## Learnings
- The viewport integration (PR #56) introduced a round-trip serialization: buffer → ANSI string → viewport → ANSI string → parse back to cells
- This round-trip exposed a latent bug in the ANSI parser that didn't affect direct rendering
- When dealing with ANSI color codes, it's important to distinguish between basic codes (30-47, 90-107) and 256-color codes (38;5;N, 48;5;N)
