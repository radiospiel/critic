# TODO

## Terminal Resize Handling

**Issue**: Window resize behavior differs between Terminal.app and iTerm2

**Current Status**:
- Using fullscreen mode (`\x1b[?1049h`) + nowrap mode (`\x1b[?7l`)
- Works perfectly in Terminal.app without requiring repaint on resize
- iTerm2 still shows visual artifacts/wrapping during resize

**Goal**: Achieve vim-like `set nowrap` behavior - no flicker or wrapping artifacts during resize

**Next Steps**:
- Investigate iTerm2-specific terminal behavior
- Consider conditional repaint logic based on terminal type detection
- Research additional escape sequences or terminal modes that iTerm2 may support
- Test with other terminal emulators (Alacritty, Kitty, WezTerm)

**References**:
- Stack Exchange discussion: https://apple.stackexchange.com/questions/144144/disable-line-wrapping-and-horizontal-scrolling-for-output-in-iterm
- Code location: `internal/app/app.go:161-164` and `internal/ui/diffview.go:281`
