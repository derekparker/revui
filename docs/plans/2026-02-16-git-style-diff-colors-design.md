# Git-Style Diff Colors Design

## Goal

Replace Chroma syntax highlighting with standard git-style diff coloring:
- Green foreground text for added lines
- Red foreground text for removed lines
- Default terminal foreground for unchanged (context) lines

## Motivation

Syntax highlighting adds visual noise to diff review. Standard git-style coloring is familiar, readable, and adapts to the user's terminal color scheme.

## Design

### Styling Changes

Replace background-color-based styles with foreground-color styles in `internal/ui/diffview.go`:

| Line Type | Current | New |
|-----------|---------|-----|
| Added | `Background(Color("22"))` green bg | `Foreground(Color("2"))` green text |
| Removed | `Background(Color("52"))` red bg | `Foreground(Color("1"))` red text |
| Context | No style | No style |

Uses standard ANSI colors (0-7) so they respect terminal themes.

### Code Removal

- Delete `internal/syntax/highlight.go` and `internal/syntax/highlight_test.go`
- Remove `Highlighter` field from `DiffViewer` struct and initialization
- Remove all `HighlightLine()` calls in rendering functions
- Remove `chroma/v2` dependency via `go mod tidy`

### What Stays the Same

- Line number styling (gray gutters)
- Hunk header styling (cyan, faint)
- Cursor styling (bold)
- Comment marker styling (yellow dot)
- Visual selection styling (dark background)
- `+`/`-`/` ` prefix characters
