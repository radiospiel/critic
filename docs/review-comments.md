# Code Review Comments - Phase 2

This document describes the code review comments feature in Critic.

## Overview

Critic now supports adding, editing, and deleting review comments on diff lines. Comments are stored in `.critic.md` files alongside your source code.

## Features

### Adding Comments

1. Navigate to a line in the diff view using `j`/`k` or arrow keys
2. Press `Enter` to open the comment input dialog
3. Type your comment (Markdown is supported)
4. Press `Ctrl+S` to save the comment
5. Press `Esc` to cancel without saving

### Editing Comments

1. Navigate to a line that has a comment (indicated by 💬)
2. Press `Enter` to edit the existing comment
3. Modify the text
4. Press `Ctrl+S` to save changes
5. Press `Esc` to cancel

### Deleting Comments

1. Navigate to a line that has a comment (indicated by 💬)
2. Press `d` to delete the comment
3. The comment is removed immediately

### Comment Indicators

Lines with comments are marked with a 💬 emoji in the diff view, making it easy to see which lines have been reviewed.

## Storage Format

Comments are stored in files with the naming pattern `.<original-filename>.critic.md`.

For example, comments on `src/main.go` are stored in `src/.main.go.critic.md`.

### File Format

Comments use a simple fence-based format:

```
--- CRITIC 3 of lines
This is a review comment.
It can span multiple lines.
Markdown is supported!
--- CRITIC END

--- CRITIC 1 of lines
Another comment on a different line.
--- CRITIC END
```

The format:
- Starts with `--- CRITIC N of lines` where N is the number of lines in the comment
- Contains the comment text (Markdown)
- Ends with `--- CRITIC END`

## Implementation Details

### Components

1. **Comment Storage** (`internal/comments/storage.go`)
   - Handles reading and writing `.critic.md` files
   - Manages comment persistence
   - Maps comments to line numbers

2. **Comment Input UI** (`internal/ui/commentinput.go`)
   - Provides a text area for entering comments
   - Supports new comments and editing existing ones
   - Keyboard shortcuts: `Ctrl+S` to save, `Esc` to cancel

3. **Diff View Integration** (`internal/ui/diffview.go`)
   - Displays comment indicators (💬)
   - Handles keyboard interactions (`Enter` to add/edit, `d` to delete)
   - Shows comment input as an overlay

### Key Bindings

In the diff view (when focused):

- `Enter` - Add a new comment or edit an existing comment on the current line
- `d` - Delete the comment on the current line (if one exists)

In the comment input dialog:

- `Ctrl+S` - Save the comment
- `Esc` - Cancel and close the dialog

## Future Enhancements

Potential improvements for Phase 3:

1. Display comment content inline or in a separate pane
2. Thread support for multiple comments per line
3. Comment resolution/approval workflow
4. Export comments to different formats (GitHub PR comments, GitLab notes, etc.)
5. Comment filtering and search
6. Collaborative commenting with author attribution
