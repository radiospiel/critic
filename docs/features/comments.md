# Comment System

The Critic comment system allows you to add and manage code review comments directly in the TUI. Comments are stored in files next to the original file and can survive file changes through intelligent diff synchronization.

## File Format

Comments are stored in two files:

1. **`<original-file>.critic.md`**: Contains the original file content with embedded CRITIC blocks
2. **`<original-file>.critic.original`**: A backup copy of the original file without comments

### CRITIC Block Format

A CRITIC block has the following structure:

```
--- CRITIC <N> lines ------------------------------
<comment line 1>
<comment line 2>
...
<comment line N>
--- CRITIC END ------------------------------
```

Example:

```
--- CRITIC 2 lines ------------------------------
This function should handle the edge case
where the input is nil
--- CRITIC END ------------------------------
```

## Usage

### Adding/Editing Comments

1. Navigate to the line you want to comment on in the diff view
2. Press `Enter` to open the comment editor
3. Type your comment
4. Press `Ctrl+S` to save (and continue editing)
5. Press `Ctrl+X` to save and exit
6. Press `Esc` to cancel without saving

### Comment Persistence

- Comments are stored in `.critic.md` and `.critic.original` files
- When the original file changes, comments are automatically repositioned based on diff analysis
- Comments on deleted lines are removed
- Comments on modified lines are preserved at the new line position

## Architecture

### Core Components

#### Comment Types (`pkg/types/comment.go`)

- `CriticBlock`: Represents a single comment block with line number and content
- `CriticFile`: Represents a file with original content and associated comments
- `CommentUpdate`: Represents a change to comments for a file

#### Diff Synchronization

The synchronization system uses `git diff` to build line mappings rather than applying patches directly, because CRITIC comment blocks embedded in the file break the context matching required by `git apply`.

- `SyncComments()`: Synchronizes comments when the original file changes
- `buildLineMappingWithGit()`: Uses `git diff --no-index --unified=0` to generate accurate line mappings
- `parseUnifiedDiff()`: Parses unified diff hunk headers to build old→new line number mappings
- Handles insertions (OldCount=0), deletions (NewCount=0), and modifications correctly
- Special handling for edge cases (insertions at file start/end)
- Comments on deleted lines are dropped; comments on unchanged lines are preserved at their new positions

**Why line mapping instead of `git apply`?**
Direct patch application with `git apply` fails because the CRITIC blocks break context matching. For example, if a diff expects consecutive lines `[line1, line2, line3]` but the file contains `[line1, CRITIC_BLOCK, line2, line3]`, git apply cannot find the context and fails.

#### UI Component (`internal/ui/commenteditor.go`)

- `CommentEditor`: Textarea-based editor for adding/editing comments
- Keyboard shortcuts: `Ctrl+S` (save), `Ctrl+X` (save & exit), `Esc` (cancel)
- Emits `CommentSavedMsg` and `CommentCancelledMsg` events

### Integration

The comment system is integrated into the main application (`internal/app/app.go`):

1. `Model` includes `commentEditor` and `commentManager` fields
2. `Enter` key activates the comment editor when in diff view
3. Comment save/cancel messages trigger file operations
4. Comment editor is rendered as an overlay when active

## Future Enhancements

- **MCP Integration**: Expose comments through Model Context Protocol
- **Comment threading**: Support for comment replies
- **Comment resolution**: Mark comments as resolved/unresolved
- **Comment export**: Export comments to various formats (JSON, Markdown, etc.)
- **Comment statistics**: Show comment counts in file list
- **Comment filtering**: Filter files by commented/uncommented
