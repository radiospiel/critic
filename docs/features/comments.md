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

#### Parser (`internal/comments/parser.go`)

- `ParseCriticFile()`: Parses a `.critic.md` file and extracts comments
- `FormatCriticFile()`: Formats a critic file with embedded comment blocks
- `ValidateCriticFile()`: Validates that a critic file is well-formed

#### File Manager (`internal/comments/filemanager.go`)

- `LoadComments()`: Loads comments from a `.critic.md` file
- `SaveComments()`: Saves comments to `.critic.md` and `.critic.original` files
- `HasComments()`: Checks if a file has comments
- `DeleteComments()`: Removes comment files

#### Diff Synchronization (`internal/comments/diffsync.go`)

- `SyncComments()`: Synchronizes comments when the original file changes using git diff
- `buildLineMappingWithGit()`: Uses `git diff --no-index` to generate accurate line mappings
- `parseUnifiedDiff()`: Parses unified diff format to build line-to-line mappings
- Handles insertions, deletions, and modifications correctly
- Special handling for edge cases (insertions at file start/end)
- Drops comments for deleted lines, preserves comments on modified lines

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

## Testing

Tests are located in `internal/comments/*_test.go`:

- **Parser tests**: Verify CRITIC block parsing and formatting
- **File manager tests**: Verify save/load/delete operations
- **Integration tests**: End-to-end workflow verification

Run tests with:

```bash
go test ./internal/comments/...
```

## Future Enhancements

- **MCP Integration**: Expose comments through Model Context Protocol
- **Comment threading**: Support for comment replies
- **Comment resolution**: Mark comments as resolved/unresolved
- **Comment export**: Export comments to various formats (JSON, Markdown, etc.)
- **Comment statistics**: Show comment counts in file list
- **Comment filtering**: Filter files by commented/uncommented
