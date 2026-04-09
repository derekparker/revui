# OSC 52 Clipboard Integration Design

**Date:** 2026-04-09  
**Status:** Approved

## Overview

Replace the current clipboard utility chain (xclip/xsel/wl-copy) with OSC 52 escape sequences for universal clipboard support across local and remote terminals.

## Goals

- Enable clipboard copying in any OSC 52-capable terminal without external dependencies
- Support SSH/remote sessions where system clipboard utilities aren't available
- Simplify clipboard implementation by removing utility detection and fallback logic
- Maintain compatibility with tmux and zellij terminal multiplexers

## Architecture

### Core Change

The `deliverToClipboard()` function in `internal/output/output.go` will write an OSC 52 escape sequence to `os.Stderr` instead of executing external clipboard utilities.

**Why stderr:** OSC 52 sequences must be written to the terminal, not stdout. Stderr is the correct destination since it's typically connected to the user's terminal while stdout might be redirected.

**Library:** Use the existing `github.com/aymanbagabas/go-osc52/v2` dependency (already in the dependency tree via Bubble Tea).

### Scope

- Replace `deliverToClipboard()` implementation
- Update success message to mention OSC 52 support
- Update help text to document terminal requirements
- Remove external utility fallback logic

## Implementation Details

### Function Implementation

**Function signature:** `deliverToClipboard(content string) (string, error)` remains unchanged.

**Implementation:**
```go
func deliverToClipboard(content string) (string, error) {
    seq := osc52.New(content)
    _, err := fmt.Fprint(os.Stderr, seq)
    if err != nil {
        return "", fmt.Errorf("failed to write OSC 52 sequence: %w", err)
    }
    return "Review copied to clipboard via OSC 52.", nil
}
```

**Import addition:** Add `github.com/aymanbagabas/go-osc52/v2` to imports in `output.go`.

### Terminal Multiplexer Support

The `go-osc52/v2` library automatically detects the `$TMUX` environment variable and wraps the OSC 52 sequence in tmux-specific escape codes. No special handling needed in our code.

Zellij passes OSC 52 sequences through to the outer terminal when configured properly.

### Size Limits

OSC 52 has terminal-dependent size limits (typically 100KB-1MB). Since code reviews are text-based and typically small, we won't implement size checking initially. If this becomes an issue in practice, we can add a size check and fallback to file output for large reviews.

### Success Feedback

OSC 52 is a write-only operation with no confirmation from the terminal. We return success immediately after writing the sequence. Users will verify it worked when they paste.

## Error Handling

### Error Conditions

- `fmt.Fprint()` fails writing to stderr (rare, would indicate terminal disconnect)
  - Return error with context
  - User sees failure message in the TUI

### No Special Handling Needed

- **Content size** - Terminals handle their own limits, worst case they truncate
- **Terminal capability** - If terminal doesn't support OSC 52, sequence is silently ignored (no harm)
- **Empty content** - Library handles it gracefully

### Removed Error Handling

- No more utility detection loop
- No more "which clipboard utility is available" logic
- Simpler error path - single point of failure instead of multiple fallbacks

## Testing Strategy

### Unit Tests (`output_test.go`)

- Test that `deliverToClipboard()` returns success (can't verify actual clipboard state)
- Test with empty content
- Test with large content (ensure no panic)

**Limitation:** Cannot unit test actual clipboard state since OSC 52 is a terminal side-effect. Automated tests verify the code doesn't error, not that clipboard actually works.

### Manual Testing Checklist

- [ ] Local terminal (verify clipboard works)
- [ ] SSH session (verify OSC 52 passes through)
- [ ] Inside tmux (verify tmux wrapping works)
- [ ] Inside zellij (verify zellij passthrough works)
- [ ] Terminal without OSC 52 support (verify graceful degradation - no error, just silent fail)

## Terminal Compatibility

### Supported Terminals

Most modern terminals support OSC 52:
- iTerm2 (macOS)
- Alacritty
- WezTerm
- kitty
- tmux (with `set-clipboard on`)
- Windows Terminal
- Konsole
- Foot

### Unsupported Terminals

Older terminals like GNOME Terminal and some terminal emulators ignore OSC 52 sequences. In these cases, the sequence is silently ignored and the clipboard won't be updated. Users can fall back to the "Write to file" option.

## Documentation Updates

### Help Text

Update the help overlay (`internal/ui/help.go`) to mention:
- Clipboard option uses OSC 52 escape sequences
- Requires OSC 52-capable terminal
- Works over SSH/remote connections

### User-Facing Changes

The "System clipboard" option label can remain unchanged. The success message will indicate "Review copied to clipboard via OSC 52" to inform users of the mechanism.

## Migration Notes

### Removed Code

- External clipboard utility detection loop (`xclip`, `xsel`, `wl-copy`)
- Stderr capture for utility errors
- Utility-specific error messages

### Added Code

- Single `osc52.New(content)` call
- Simplified error handling

### Dependencies

No new dependencies - `github.com/aymanbagabas/go-osc52/v2` is already in the dependency tree.

## Success Criteria

- [ ] Clipboard copying works in local OSC 52-capable terminal
- [ ] Clipboard copying works over SSH session
- [ ] Clipboard copying works inside tmux
- [ ] Clipboard copying works inside zellij
- [ ] Code is simpler than previous utility-based implementation
- [ ] No new dependencies added
- [ ] Tests pass
