# Reorganize Source Directories

**Started:** 2026-01-26 12:58:20
**Ended:** 2026-01-26 13:02:00
**Complexity:** Simple
**Outcome:** Success

## Task Description
Reorganize source code structure:
- Move `cmd/critic/` → `src/cmd`
- Move `internal/(foo)` → `src/foo`
- Move `pkg/` → `src/pkg`
- Adjust Makefiles and import paths

## Strategy
**Refactoring** - This is a code restructuring task.

## Progress

### Step 1: Analyze current structure
- [x] Identified directories to move
- [x] Found Makefiles to update
- [x] Reviewed go.mod (module: git.15b.it/eno/critic)

### Step 2: Move directories
- [x] Create src/ directory
- [x] Move cmd/critic/ → src/cmd
- [x] Move internal/* → src/*
- [x] Move pkg/ → src/pkg

### Step 3: Update imports
- [x] Update all import paths from internal/* to src/*
- [x] Update all import paths from pkg/* to src/pkg/*

### Step 4: Update build
- [x] Update Makefile build path

### Step 5: Verify
- [x] Run go build
- [x] Run tests

## Obstacles
None encountered - straightforward refactoring.

## Notes
Module path is `git.15b.it/eno/critic`, so imports will change from:
- `git.15b.it/eno/critic/internal/foo` → `git.15b.it/eno/critic/src/foo`
- `git.15b.it/eno/critic/pkg/foo` → `git.15b.it/eno/critic/src/pkg/foo`
