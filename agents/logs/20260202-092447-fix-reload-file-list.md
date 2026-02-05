# Fix File List Reload on Git Changes

**Started:** 2026-02-02 09:24
**Ended:** 2026-02-02 09:26
**Complexity:** Simple
**Outcome:** Fixed

## Task
When a new file is created or added to git, the frontend receives a reload event but doesn't reload the file list.

## Strategy
Bug Fix - Something is broken (reload event received but file list doesn't update)

## Root Cause Analysis
Found by exploration:
1. **App.tsx** has a `files` state and correctly calls `loadFileList()` when reload event received
2. **FileList.tsx** has its **own independent `files` state** fetched once on mount
3. **App.tsx doesn't pass `files` as a prop to FileList** - so updates never reach FileList

## Solution
1. Pass `files` from App.tsx to FileList.tsx as a prop
2. Remove FileList's independent state and useEffect that fetches files
3. FileList should use the passed `files` prop instead

## Changes Made
- **FileList.tsx**: Added `files` prop to interface, removed local `files` state, updated useEffect to not fetch files
- **App.tsx**: Pass `files` prop to FileList component

## Progress
- [x] Identify root cause
- [x] Read App.tsx and FileList.tsx
- [x] Implement fix
- [x] Build succeeds
- [x] Tests pass
- [ ] Manual test
- [ ] Commit

## Obstacles
(None)
