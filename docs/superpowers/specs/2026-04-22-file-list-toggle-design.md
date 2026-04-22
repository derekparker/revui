# File List Toggle — Design Spec

**Date:** 2026-04-22  
**Status:** Approved

## Overview

Add a `ctrl+h` keybinding to toggle the file list panel on the left side of the UI. When hidden, the diff viewer expands to the full terminal width. The toggle is stateful and can be used any number of times during a review session.

## State Changes

Add `hideFileList bool` to `RootModel` in `internal/ui/root.go`.

Add a helper method to centralize the diff viewer width calculation:

```go
func (m RootModel) diffViewerWidth() int {
    if m.hideFileList {
        return m.width
    }
    return m.width - m.fileListWidth - 3
}
```

This replaces the inline expression `m.width - m.fileListWidth - 3` in two places:
1. `WindowSizeMsg` handling in `Update()`
2. `View()` when computing the diff panel width

## Key Handling

In `handleKeyMsg` in `root.go`, add a `ctrl+h` case:

```go
case "ctrl+h":
    m.hideFileList = !m.hideFileList
    if m.hideFileList && m.focus == focusFileList {
        m.focus = focusDiffViewer
    }
    m.diffViewer.SetSize(m.diffViewerWidth(), m.height-2)
    return m, nil
```

Behavior:
- Toggles `hideFileList` on each press.
- If the file list is being hidden and focus is currently on the file list, focus automatically shifts to the diff viewer.
- Immediately resizes the diff viewer to use the newly available width.

Also guard the existing `h` key handler (move focus left to file list) so it is a no-op when `hideFileList` is true:

```go
case "h":
    if m.focus == focusDiffViewer && !m.hideFileList {
        m.focus = focusFileList
    }
    return m, nil
```

## View Rendering

In `RootModel.View()`, conditionally skip rendering the file list panel:

**When `hideFileList` is false** (current behavior): render `fileListPanel` and `diffPanel` joined horizontally.

**When `hideFileList` is true**: skip `fileListPanel` entirely. Render only `diffPanel` with `Width(m.width)`. The `lipgloss.JoinHorizontal` call becomes just the diff panel (or is replaced with a direct write).

The file list model retains its full state (cursor position, file list) while hidden, so restoring the panel shows it exactly as it was.

## Help Overlay

Add one line in `help.go` under the "Views" section:

```
  Ctrl+h      Toggle file list
```

## Scope

- No changes to `FileList`, `DiffViewer`, or any other sub-model.
- No animation or smooth transition — the panel appears/disappears immediately.
- File navigation state (cursor position) is preserved while the file list is hidden.
- The status bar hint text is not changed (the help overlay is the documentation).
