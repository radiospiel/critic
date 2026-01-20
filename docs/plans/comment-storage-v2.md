# Comment Storage v2 Design Document

## Overview

This document describes the redesign of how comments are stored and managed in the critic application. The main changes are:
1. Store actual git commit SHA1s instead of symbolic references
2. Allow commenting on any line (deleted, added, or unchanged)
3. Store code context with each comment for better code review persistence

## Current Behavior

Comments can only be added on **added lines** (lines with `+` prefix in the diff). The comment is stored with:
- `line_number`: The line number in the file
- `code_version`: A git ref (often a symbolic name like a branch)

## New Behavior

Comments can be added on **any line** (deleted `-`, added `+`, or unchanged ` `).

### Database Schema Changes

Rename columns for clarity:
- `line_number` → `lineno`
- `code_version` → `commit`

Add new column:
- `context` (TEXT): The code context around the commented line

### LineDisplacement Changes

The `LineDisplacement` struct will store the actual SHA1 commit hashes:

```go
type LineDisplacement struct {
    blocks      []LineDisplacementBlock
    ref1SHA1    string  // Actual SHA1 of the "old" commit
    ref2SHA1    string  // Actual SHA1 of the "new" commit
}
```

When building the displacement map, symbolic refs (like branch names) are resolved to their SHA1 commits using `git rev-parse`.

### Comment Storage Logic

When the user adds a comment on a line:

1. **Determine which commit SHA1 to store:**
   - For **deleted lines**: Use `ref1SHA1` (the "old" version)
   - For **added or unchanged lines**: Use `ref2SHA1` (the "new" version)

2. **Determine which line number to store:**
   - For **deleted lines**: Use `firstOldLine` from the diff block
   - For **added or unchanged lines**: Use `firstNewLine` from the diff block

3. **Store the code context:**
   - For **deleted or added lines**: Read from the `txtWithContext` field in the `LineDisplacementBlock`
   - For **unchanged lines**: Read via `git show <commit>:<file>` and extract `2*contextLines + 1` lines (context lines before and after)

### Context Reading

#### For Deleted/Added Lines

The context is already computed in `LineDisplacementBlock.txtWithContext` during the displacement map building:

```go
// In populateBlockContent, for blocks with firstOldLine:
startLine := block.firstOldLine - contextLines
endLine := block.firstOldLine + block.numOfLInes + contextLines
block.txtWithContext = readLinesFromGitShow(path, ref, startLine, endLine)
```

For added lines, we need to look up the corresponding `move` block or compute similarly.

#### For Unchanged Lines

For unchanged lines not in any displacement block, read the context on-demand:

```bash
git show <commit>:<file> | sed -n '<lineno-context>,<lineno+context>p'
```

## Implementation Plan

1. **Update `LineDisplacement` struct**
   - Add `ref1SHA1` and `ref2SHA1` fields
   - Resolve symbolic refs to SHA1s in `BuildLineDisplacement`

2. **Database schema migration**
   - Add migration from schema v1 to v2
   - Rename `line_number` → `lineno`
   - Rename `code_version` → `commit`
   - Add `context` column

3. **Update `Message` struct and database operations**
   - Update field names (`LineNumber` → `Lineno`, `CodeVersion` → `Commit`)
   - Add `Context` field
   - Update all SQL queries

4. **Update comment creation flow**
   - Determine correct commit SHA1 based on line type
   - Determine correct line number based on line type
   - Fetch/store context for all comment types

## Data Flow

```
User hits Enter on a line
    ↓
Check line type (deleted/added/unchanged)
    ↓
Get SHA1 from LineDisplacement (ref1SHA1 or ref2SHA1)
    ↓
Get line number from diff (old or new)
    ↓
Get context (from block.txtWithContext or git show)
    ↓
Store in database (lineno, commit, context)
```

## Examples

### Comment on Deleted Line

```
diff:
  -5  old line 5    <- User comments here
  +6  new line 6

Stored:
  lineno: 5
  commit: <ref1SHA1>  # The old commit
  context: "...\nold line 2\nold line 3\nold line 4\nold line 5\nold line 6\nold line 7\nold line 8\n..."
```

### Comment on Added Line

```
diff:
  -5  old line 5
  +6  new line 6    <- User comments here

Stored:
  lineno: 6
  commit: <ref2SHA1>  # The new commit
  context: "...\nold line 2\nold line 3\nnew line 4\nnew line 5\nnew line 6\nnew line 7\n..."
```

### Comment on Unchanged Line

```
diff:
   4  unchanged line 4
   5  unchanged line 5    <- User comments here
   6  unchanged line 6

Stored:
  lineno: 5
  commit: <ref2SHA1>  # The new commit
  context: "...\nunchanged line 2\nunchanged line 3\nunchanged line 4\nunchanged line 5\nunchanged line 6\nunchanged line 7\n..."
```
