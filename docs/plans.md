# Plans and Roadmap

## Completed

- Phase 1: Basic diff viewer with syntax highlighting
- Phase 2: Code review comments with SQLite storage
- Phase 3: MCP server integration for AI assistants
- Web UI with htmx and WebSocket updates

## Planned Features

### File System Event Handling

1. **Phase 1:** Integrate file watcher into app for automatic refresh on file changes
2. **Phase 2:** Watch `.git/` directory for ref changes (HEAD, branches, tags)
3. **Phase 3:** Optimized refresh - reload only changed files, not entire diff

### Web UI Improvements

- Split-view for side-by-side diff comparison
- File tree view with folder collapsing
- Search functionality across files and comments
- Comment threading UI improvements
- Mobile-responsive design

### TUI Improvements

- Configurable color schemes
- Custom keybinding support
- Inline diff folding

## Contributing

See [hacking.md](hacking.md) for development setup and testing guidelines.
