# Auto-Refresh for Uncommitted Changes Mode

## Problem

When revui is open in a long-running tmux split in uncommitted changes mode, the file list and diffs are static — loaded once at startup. Users want to see changes reflected in real time as they edit files, without restarting revui.

This only applies to uncommitted changes mode. In branch diff mode, all changes are committed and no refresh is expected.

## Design

### Approach: Tick + Async Commands (Bubble Tea native)

Use `tea.Tick` to poll every 2 seconds, spawning an async `tea.Cmd` that fetches the latest file list and current diff from git. Results arrive as a message and update the UI without blocking.

### Message Flow

```
tea.Tick (2s) → tickRefreshMsg
    ↓
RootModel.Update():
    → If refreshInProgress, schedule next tick and skip
    → Otherwise, set refreshInProgress = true
    → Return tea.Cmd that calls UncommittedFiles() + UncommittedFileDiff()
    ↓
Cmd completes → refreshResultMsg{files, currentDiff, err}
    ↓
RootModel.Update():
    → Clear refreshInProgress
    → Compare new file list with current
    → Update file list and diff if changed
    → Schedule next tick
```

### New Types

- `tickRefreshMsg` — empty struct, signals time to poll
- `refreshResultMsg` — carries `[]git.ChangedFile`, `*git.FileDiff` for current file, and error

### New RootModel Fields

- `refreshInProgress bool` — prevents overlapping git operations

### Change Detection

- **File list**: Compare new `[]ChangedFile` paths and statuses against current list. Any difference triggers an update.
- **Current diff**: Re-fetch diff for the currently viewed file alongside the file list (one async operation, one result message).

### State Preservation

- **Cursor**: Preserved if the selected file still exists in the new list. If the file was removed, cursor moves to the nearest neighbor.
- **Scroll**: Scroll offset preserved, clamped to new diff bounds if the file got shorter.
- **Comments**: Left as-is. No orphan detection in v1.

### File List Updates

- Files removed from the working tree (reverted, deleted) disappear from the list automatically.
- New files appear in the list automatically.
- FileList needs a method to update its items and adjust the cursor.

### Scope

- Only active in `modeUncommitted`. In `modeBranch`, no tick is scheduled and there is zero overhead.
- `Init()` returns the first tick command when in uncommitted mode.
- `loadFileDiff()` is reused unchanged.

### What This Does NOT Include

- Filesystem watching (fsnotify) — polling is simpler and reliable
- Configurable poll interval — 2 seconds is a reasonable fixed default
- Orphaned comment detection — comments stay at their line numbers, user manages manually
- Any changes to branch diff mode
