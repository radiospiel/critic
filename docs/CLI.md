# Critic CLI Documentation

## Overview

Critic is a git diff viewer with syntax highlighting and interactive navigation. It allows you to compare changes across multiple base points and cycle between them interactively.

## Command-Line Syntax

```bash
critic [options] [bases..current] [-- paths...]
```

### Arguments

#### Base and Current Specification

The `bases..current` argument specifies what to compare:

- **bases**: Comma-separated list of git references to use as comparison points
  - Can be branch names (e.g., `main`, `develop`, `origin/main`)
  - Can be commit SHAs (e.g., `a1b2c3d`)
  - Can be the special value `merge-base` which auto-resolves to the merge base with main/master
- **current**: The target reference to compare against
  - Use `current` for the working directory (includes uncommitted changes)
  - Can be any git reference (branch, tag, commit SHA)

**Examples:**
```bash
# Compare working directory against merge-base
critic merge-base..current

# Compare against multiple bases
critic merge-base,origin/main,HEAD..current

# Compare specific commit range
critic main,develop..v1.0.0

# Just specify bases (current defaults to working directory)
critic main,develop
```

If no bases are specified, Critic automatically determines defaults:
1. `merge-base` - the merge base with main/master
2. `origin/<current-branch>` - if it exists
3. `HEAD` - the last commit

#### Extension Filtering

Use `--extensions` to filter files by extension:

```bash
critic --extensions=go,rs main..current
```

Without this flag, Critic includes a comprehensive list of default extensions (see `internal/config/extensions.go`).

#### Path Filtering

Specify paths after `--` to limit the diff to specific directories or files:

```bash
critic main..current -- src tests
critic --extensions=go main..current -- internal/
```

### Complete Examples

```bash
# View all changes in working directory since merge-base
critic

# View only Go and Rust files
critic --extensions=go,rs

# Compare specific paths against main branch
critic main..current -- internal/ cmd/

# Multiple bases with extension and path filtering
critic --extensions=c,cpp,h merge-base,origin/main..current -- src/
```

## Interactive Navigation

Once Critic is running, use these keyboard shortcuts:

- **b** - Cycle through base references (changes which base you're comparing against)
- **Tab** - Switch focus between file list and diff view
- **↑/↓** or **j/k** - Navigate up/down
- **Space** - Page down in diff view
- **Shift+Space** - Page up in diff view
- **q** or **Ctrl+C** - Quit

## Status Bar

The status bar at the bottom shows:

```
Base: merge-base → current • Branch: feature/xyz • Files: 5 • b: base • Tab: switch • ?: help • q: quit
```

- **Base**: Shows current base reference and target (format: `base → target`)
- **Branch**: Current git branch
- **Files**: Number of files in the diff
- **Keyboard hints**: Available shortcuts

## Base Resolution and Polling

### Automatic Resolution

- The special value `merge-base` automatically finds the merge base with your main/master branch
- All bases are resolved to commit SHAs at startup
- You can still use friendly names in the UI

### Automatic Polling

Critic automatically polls git every 10 seconds to detect changes in:
- The merge base (in case main/master has moved)
- Remote branches (in case origin/... has updated)
- Local branches (in case they've been updated)

When a change is detected, the diff automatically refreshes to show the latest changes.

## Untracked Files

When the target is `current` (working directory), Critic includes untracked files:
- Files are discovered using `git ls-files --others --exclude-standard`
- This respects `.gitignore` (shows untracked files, not ignored files)
- Untracked files are diffed against an empty state (all content shown as additions)
- Extension filtering applies to untracked files as well

When the target is a specific git reference (not `current`), only git-tracked changes are shown.

## File Extensions

Default file extensions include common programming languages:
- **Languages**: Go, Rust, C/C++, JavaScript/TypeScript, Python, Ruby, Java, Kotlin, C#, PHP, Shell, Perl, Lua, Elixir, Erlang, Haskell, Swift, Scala, R, Julia
- **Web**: HTML, CSS, SCSS, Vue, Svelte
- **Config**: YAML, TOML, JSON, XML, INI
- **Docs**: Markdown, reStructuredText, text
- **Data**: SQL, Protobuf

For the complete list, see `internal/config/extensions.go`.

## Error Handling

If an error occurs (e.g., invalid git reference, failed to resolve merge-base), Critic will:
1. Print an error message to stderr
2. Exit with a non-zero status code

Example errors:
```bash
# Invalid base reference
$ critic nonexistent..current
Error: failed to resolve base nonexistent: ...

# Not in a git repository
$ critic
Error: Not a git repository
```

## Advanced Usage

### Comparing Against Multiple Points

Compare your work against merge-base, the remote branch, and HEAD all at once:

```bash
critic merge-base,origin/feature-branch,HEAD..current
```

Press `b` to cycle through each base and see how your changes look from different perspectives.

### Language-Specific Reviews

Review only Python changes:

```bash
critic --extensions=py merge-base..current
```

### Subset Reviews

Review changes in a specific directory:

```bash
critic merge-base..current -- internal/app/
```

---

*This documentation was generated as part of a Claude Code run on 2025-12-28.*
