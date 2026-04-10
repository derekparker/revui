# Long File Path Wrapping Design

**Date:** 2026-04-10  
**Status:** Approved

## Overview

Improve file list rendering to gracefully wrap long file paths across multiple lines with proper indentation alignment, instead of wrapping underneath the selection arrow and status icon.

## Problem

Currently, long file paths in the sidebar wrap to a new line directly underneath the selection indicator (`▸`) and status icon (`M`), making them hard to read:

```
▸ M src/very/long/path/to/component/
file.tsx
```

The wrapped portion should remain aligned with the first line's text.

## Goals

- Wrap long paths across multiple lines when they exceed sidebar width
- Indent continuation lines to align with the path text (not the prefix)
- Maintain visual hierarchy and selection styling
- Use existing lipgloss dependency for width handling

## Architecture

### Core Change

Modify `FileList.View()` in `internal/ui/filelist.go` to apply lipgloss width constraints and indentation to file paths after rendering the prefix (arrow + status icon).

### Approach: Lipgloss MaxWidth Styling

Use lipgloss's built-in `Width()` styling to constrain text width and enable wrapping, combined with `PaddingLeft()` for continuation line indentation.

**Why lipgloss:**
- Already in dependency tree (Bubble Tea uses it)
- Automatically handles ANSI codes when measuring width
- Provides built-in wrapping and padding primitives
- No new dependencies needed

## Implementation Details

### Current Rendering Flow

```go
// Line 76-91 in filelist.go
for i, f := range fl.files {
    icon := statusIcon(f.Status)
    line := icon + " " + f.Path

    if i == fl.cursor {
        line = selectedStyle.Render("▸ " + line)
    } else {
        line = unselectedStyle.Render("  " + line)
    }
    b.WriteString(line)
    b.WriteByte('\n')
}
```

### New Rendering Flow

1. **Calculate prefix and available width:**
   - Prefix for selected: `"▸ "` (2) + status icon (1) + space (1) = 4 chars
   - Prefix for unselected: `"  "` (2) + status icon (1) + space (1) = 4 chars
   - Available path width: `fl.width - 4`

2. **Create path wrapping style:**
   ```go
   pathStyle := lipgloss.NewStyle().
       Width(availableWidth).
       PaddingLeft(4)  // Indent continuation lines to align
   ```

3. **Render path with wrapping:**
   - Apply `pathStyle` to the file path
   - Lipgloss handles line wrapping at width boundary
   - Continuation lines get 4 spaces padding automatically

4. **Combine prefix with wrapped path:**
   - Render wrapped path to string
   - Split into lines
   - First line: prepend prefix (`"▸ M "` or `"  M "`)
   - Continuation lines: already have padding from style

5. **Apply selection styling:**
   - Apply selection style (bold/color) to entire rendered output
   - Maintains visual hierarchy across all lines

### Modified View() Function

**Key changes:**
- Lines 76-91: Replace simple string concatenation with styled path wrapping
- Add logic to handle multi-line path rendering
- Apply selection styling to complete multi-line output

**Files modified:**
- `internal/ui/filelist.go` - Update `View()` method

**No new files created.**

## Width Handling

### Available Width Calculation

```go
availableWidth := fl.width - 4
```

Where 4 accounts for:
- Selection indicator: 2 chars (`"▸ "` or `"  "`)
- Status icon: 1 char (`"M"`, `"A"`, `"D"`, etc.)
- Spacing: 1 char

### Edge Cases

**Extremely narrow width (< 10 characters):**
- Still apply wrapping, may result in many short lines
- Better than no wrapping at all

**Exact boundary width:**
- Lipgloss handles this - will wrap at exact width

**Empty or single file:**
- No special handling needed, existing logic works

## Visual Examples

### Before (Current)
```
▸ M src/internal/very/long/path/to/
component/file.tsx
  A another/file.go
```

### After (With Wrapping)
```
▸ M src/internal/very/long/path/to/
    component/file.tsx
  A another/file.go
```

## Testing Strategy

### Manual Testing

**Test cases:**
1. **Short paths** - Verify no wrapping when path fits
2. **Medium paths** - Verify wraps to 2 lines with correct indent
3. **Very long paths** - Verify wraps to 3+ lines, all indented
4. **Narrow terminal** - Resize to < 40 columns, verify wrapping works
5. **Different statuses** - Test M, A, D, R, B status icons
6. **Selection changes** - Verify styling applies to all wrapped lines

**Manual test procedure:**
1. Create test files with various path lengths
2. Run revui in different terminal widths
3. Navigate through file list (j/k keys)
4. Verify visual alignment and indentation
5. Check selection highlighting on wrapped paths

### Edge Case Validation

- Path exactly at width boundary
- Very narrow sidebar (< 20 chars)
- Empty file list
- Single character available width

## Success Criteria

- [ ] Long paths wrap to multiple lines
- [ ] Continuation lines indent to align with first line text
- [ ] Selection styling applies to all lines of wrapped path
- [ ] Works across different terminal widths
- [ ] No visual regression for short paths
- [ ] All file status icons render correctly
- [ ] Code compiles and runs without errors

## Migration Notes

### Breaking Changes

None - this is a visual enhancement only.

### Backward Compatibility

Fully backward compatible. Short paths that fit on one line will render identically to before.

### Dependencies

No new dependencies. Uses existing `github.com/charmbracelet/lipgloss` from Bubble Tea.
