# Long File Path Wrapping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wrap long file paths in the sidebar across multiple lines with proper indentation alignment using lipgloss width constraints.

**Architecture:** Modify `FileList.View()` to apply lipgloss `Width()` styling for text wrapping, then manually add indentation to continuation lines for proper alignment with the first line.

**Tech Stack:** Go 1.25, lipgloss (existing dependency)

---

## File Structure

**Modified files:**
- `internal/ui/filelist.go` - Update `View()` method (lines 69-93)

**No new files created.**

---

### Task 1: Implement path wrapping with lipgloss

**Files:**
- Modify: `internal/ui/filelist.go:69-93`

- [ ] **Step 1: Update View() method with path wrapping logic**

Replace the View() method in `internal/ui/filelist.go` (lines 69-93) with:

```go
// View renders the file list.
func (fl FileList) View() string {
	if len(fl.files) == 0 {
		return "No changed files"
	}

	var b strings.Builder
	for i, f := range fl.files {
		icon := statusIcon(f.Status)
		
		// Calculate available width for path (sidebar width - prefix length)
		// Prefix: "▸ " (2) + icon (1) + " " (1) = 4 chars
		availableWidth := fl.width - 4
		if availableWidth < 1 {
			availableWidth = 1 // Minimum width
		}
		
		// Create wrapping style for path
		pathStyle := lipgloss.NewStyle().Width(availableWidth)
		wrappedPath := pathStyle.Render(f.Path)
		
		// Split into lines
		lines := strings.Split(wrappedPath, "\n")
		
		// Build the complete entry with proper prefixes
		for lineIdx, pathLine := range lines {
			var prefix string
			if lineIdx == 0 {
				// First line: add arrow/indent + icon + space
				if i == fl.cursor {
					prefix = "▸ " + icon + " "
				} else {
					prefix = "  " + icon + " "
				}
			} else {
				// Continuation lines: indent to align with path text
				prefix = "    " // 4 spaces to align with first line text
			}
			
			line := prefix + pathLine
			
			// Apply selection styling
			if i == fl.cursor {
				if fl.focused {
					line = selectedStyle.Render(line)
				} else {
					line = selectedUnfocusedStyle.Render(line)
				}
			}
			
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
```

- [ ] **Step 2: Build to verify compilation**

Run: `go build ./cmd/revui`

Expected: Successful build, no errors

- [ ] **Step 3: Run existing tests**

Run: `go test ./internal/ui/ -v`

Expected: All tests PASS (or note any visual output changes in View() tests)

- [ ] **Step 4: Manual testing - Create test files**

Create test files with various path lengths:

```bash
# Short path (fits on one line)
mkdir -p test-paths
touch test-paths/short.go

# Medium path (wraps to 2 lines)
mkdir -p test-paths/some/moderately/long/directory/path
touch test-paths/some/moderately/long/directory/path/medium.go

# Long path (wraps to 3+ lines)
mkdir -p test-paths/this/is/a/very/long/path/that/should/definitely/wrap/across/multiple/lines
touch test-paths/this/is/a/very/long/path/that/should/definitely/wrap/across/multiple/lines/long.go

# Stage all for git
git add test-paths/
```

- [ ] **Step 5: Manual testing - Test at different widths**

Run revui and verify wrapping behavior:

```bash
# Build fresh binary
go build ./cmd/revui

# Run in terminal (resize to different widths and verify):
./revui

# Test cases:
# 1. Normal width (80+ cols) - verify short paths fit on one line
# 2. Medium width (40-60 cols) - verify medium paths wrap to 2 lines
# 3. Narrow width (20-30 cols) - verify long paths wrap to 3+ lines
# 4. Very narrow (< 20 cols) - verify still works without crashing

# Verify for each case:
# - Continuation lines are indented 4 spaces
# - All lines align properly
# - Selection highlighting (▸) works on wrapped paths
# - Different status icons (M, A, D) display correctly
# - Navigation (j/k) works across wrapped entries
```

Expected:
- Short paths render on single line (no change from before)
- Medium paths wrap to 2 lines with continuation indented
- Long paths wrap to 3+ lines, all continuations indented
- Selection styling applies to all lines of wrapped path
- Visual alignment is consistent

- [ ] **Step 6: Clean up test files**

```bash
# Remove test files
git reset HEAD test-paths/
rm -rf test-paths/
```

- [ ] **Step 7: Commit**

```bash
git add internal/ui/filelist.go
git commit -m "feat: wrap long file paths with proper indentation

Implement multi-line wrapping for long file paths in the sidebar
using lipgloss Width() styling. Continuation lines are indented
4 spaces to align with the first line text.

- Calculate available width (sidebar width - 4 chars for prefix)
- Apply lipgloss Width() to wrap paths at boundary
- Manually indent continuation lines for alignment
- Apply selection styling to all lines of wrapped paths

Works across different terminal widths and file status types.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Manual Testing Checklist

After implementation, verify these scenarios:

### Basic Functionality
- [ ] Short paths (< available width) render on single line
- [ ] Medium paths wrap to 2 lines with correct indentation
- [ ] Very long paths wrap to 3+ lines, all indented consistently
- [ ] Empty file list shows "No changed files"

### Width Variations
- [ ] Wide terminal (100+ cols) - short wrapping
- [ ] Normal terminal (80 cols) - moderate wrapping
- [ ] Narrow terminal (40 cols) - heavy wrapping
- [ ] Very narrow terminal (< 30 cols) - extreme wrapping but no crash

### File Status Icons
- [ ] Modified files (M) display correctly on wrapped paths
- [ ] Added files (A) display correctly on wrapped paths
- [ ] Deleted files (D) display correctly on wrapped paths
- [ ] Renamed files (R) display correctly on wrapped paths
- [ ] Binary files (B) display correctly on wrapped paths

### Selection and Navigation
- [ ] Selection indicator (▸) shows on first line of wrapped path
- [ ] Selection highlighting applies to ALL lines of wrapped entry
- [ ] Unfocused selection styling works on wrapped paths
- [ ] j/k navigation works correctly across wrapped entries
- [ ] G (jump to bottom) works with wrapped entries
- [ ] gg (jump to top) works with wrapped entries

### Edge Cases
- [ ] Single file in list (no special issues)
- [ ] All files have short paths (no wrapping needed)
- [ ] All files have long paths (heavy wrapping throughout)
- [ ] Path exactly at width boundary (wraps correctly)
- [ ] Extremely narrow width (< 10 cols) - degrades gracefully

## Success Criteria

- [ ] Code compiles successfully
- [ ] No test regressions
- [ ] Long paths wrap to multiple lines
- [ ] Continuation lines indented 4 spaces
- [ ] Visual alignment is clean and consistent
- [ ] Selection styling works on all lines
- [ ] Works across different terminal widths
- [ ] All status icons render correctly
- [ ] Navigation works as expected
