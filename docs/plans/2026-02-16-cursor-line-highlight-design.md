# Cursor Line Highlight Design

## Goal

Add a subtle background tint to the cursor line in the diff viewer so the current line is visually distinct without being distracting.

## Motivation

Currently the cursor is indicated only by a bold `→ ` prefix. In dense diffs it can be hard to track which line has focus. A subtle background tint (like vim's `cursorline`) makes the cursor position immediately obvious.

## Design

### New Style

Add `cursorLineStyle` to the existing style block in `internal/ui/diffview.go`:

```go
cursorLineStyle = lipgloss.NewStyle().Background(lipgloss.Color("236"))
```

ANSI color 236 is a very dark gray — one shade above terminal default black. It provides a visible-but-subtle distinction.

### Application

In `View()`, when `isCursor` is true, wrap the already-rendered line in `cursorLineStyle.Render(line)` before prepending the `→ ` prefix. This covers code content, line numbers, and gutter with the background tint.

Applies identically to both unified and side-by-side render paths since wrapping happens in `View()` after both render functions return.

### What Stays the Same

- Green/red foreground colors preserved (background tint doesn't interfere with foreground)
- Bold `→ ` prefix remains
- Visual selection styling unchanged
- Hunk header cursor highlighting works the same way
- All navigation behavior unchanged

### Interaction with Visual Mode

When a line is both the cursor and in visual selection, the visual selection background (color 238) takes precedence since it's applied first and the cursor prefix distinguishes the cursor line within the selection.
