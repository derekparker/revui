# Output Delivery Design

**Goal:** Replace the hard-coded clipboard-only output with a selection screen that lets the user choose where to send their review: to a running Claude Code instance, tmux paste buffer, system clipboard, or a file.

**Motivation:** The current `clipboard.WriteAll()` approach fails silently in TTY/SSH sessions without a display server. Since revui is commonly used alongside Claude Code in tmux, direct delivery to a Claude pane is the ideal workflow.

## Architecture

Uses Approach A: a new `focusOutputSelect` focus area in `RootModel` with an `OutputSelector` sub-model, following the existing focus-routing pattern (FileList, DiffViewer, CommentInput).

When `ZZ` is pressed, instead of immediately calling `tea.Quit`, RootModel formats comments, detects available targets, initializes the `OutputSelector`, and transitions focus. The TUI remains active until the user picks a destination or cancels.

## Claude Instance Detection

1. Check `$TMUX` — skip detection if not in tmux
2. Run `tmux list-panes -a -F '#{session_name}:#{window_index}.#{pane_index} #{pane_current_command} #{pane_pid}'`
3. Filter panes where `pane_current_command == "claude"`
4. Exclude the current pane (via `$TMUX_PANE`)
5. Each match becomes a selectable `OutputTarget` with `Kind = targetClaude`

Detection runs once on `ZZ` press. No background polling.

## OutputSelector Component (`internal/ui/outputselect.go`)

### Types

```go
type targetKind int

const (
    targetClaude targetKind = iota
    targetTmuxBuffer
    targetClipboard
    targetFile
)

type OutputTarget struct {
    Kind       targetKind
    Label      string   // display name, e.g. "revui:0.0  claude"
    TmuxTarget string   // pane identifier for send-keys (Claude targets only)
}
```

### Item Ordering

1. Detected Claude instances (if any)
2. Separator (`-- or --`)
3. "tmux paste buffer" (only if `$TMUX` is set)
4. "System clipboard"
5. "Write to file"

### Keys

- `j/k` or arrows: navigate
- `Enter`: select target
- `q`: cancel (quit without sending)

### Rendering

Full-screen centered list. Title: "Send review to:". Footer: keybinding hints. Highlighted item uses project accent color. Separator is non-selectable.

## Delivery Logic (`internal/output/`)

`Deliver(target OutputTarget, content string) (string, error)` — returns a status message or error.

| Target | Action |
|--------|--------|
| `targetClaude` | Write to `/tmp/revui-review-<unix-ts>.md`, then `tmux send-keys -t <pane> '@/tmp/revui-review-<ts>.md '` (no Enter — user adds context before submitting) |
| `targetTmuxBuffer` | `tmux load-buffer -` with content on stdin |
| `targetClipboard` | `clipboard.WriteAll(content)` |
| `targetFile` | Write to `/tmp/revui-review-<unix-ts>.md` |

## RootModel Integration

1. Add `focusOutputSelect` to `focusArea` enum
2. Add `outputSelector OutputSelector` field
3. `ZZ` press: format comments, detect targets, init `OutputSelector`, set focus
4. On selection message: execute `Deliver()`, store result, `tea.Quit`
5. On delivery error: show error in status area, stay on selection screen
6. `View()` renders `OutputSelector` full-screen when focused (replaces diff/filelist)

## main.go Changes

Remove `clipboard.WriteAll()` logic. After TUI exits, print `rm.DeliveryResult()` — a human-readable status message (e.g. "Review sent to Claude at revui:0.0" or "Review written to /tmp/revui-review-1740412800.md").

## Error Handling

- tmux detection failure: silently skip Claude instances, show fallbacks only
- Any delivery failure: show error, stay on selection screen so user can pick another option
- No comments: skip selector entirely, quit directly
- `q` on selector: quit without sending
