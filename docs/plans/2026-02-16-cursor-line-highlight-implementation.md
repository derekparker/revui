# Cursor Line Highlight Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a subtle dark background tint to the cursor line in the diff viewer so it's visually distinct without overpowering the red/green diff colors.

**Architecture:** Add a single new lipgloss style (`cursorLineStyle`) and apply it in `View()` by wrapping the rendered line content when `isCursor` is true. The change is entirely within `internal/ui/diffview.go`.

**Tech Stack:** Go, lipgloss (already imported)

---

### Task 1: Add cursor line background style and apply it

**Files:**
- Modify: `internal/ui/diffview.go:13-22` (style block)
- Modify: `internal/ui/diffview.go:326-336` (View cursor rendering)
- Test: `internal/ui/diffview_test.go`

**Step 1: Write the failing test**

Add to `internal/ui/diffview_test.go`:

```go
func TestDiffViewCursorLineHighlight(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())

	// Cursor starts at line 0 (hunk header)
	view := dv.View()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")

	// The cursor line (first line) should have the → prefix
	if !strings.Contains(lines[0], "→") {
		t.Error("cursor line should contain → prefix")
	}

	// The cursor line should have ANSI background escape sequence
	// ANSI 236 background = ESC[48;5;236m
	if !strings.Contains(lines[0], "\033[") {
		t.Error("cursor line should contain ANSI styling for background highlight")
	}

	// Non-cursor lines should NOT have the background
	if strings.Contains(lines[1], "→") {
		t.Error("non-cursor line should not contain → prefix")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/ -run TestDiffViewCursorLineHighlight -v`
Expected: FAIL — cursor line currently has bold styling but no background escape sequence for color 236.

**Step 3: Add the style and apply it**

In `internal/ui/diffview.go`, add to the style var block (after `cursorStyle` on line 18):

```go
cursorLineStyle = lipgloss.NewStyle().Background(lipgloss.Color("236"))
```

In `View()`, change the cursor rendering block (lines 330-336) from:

```go
if isCursor {
    line = cursorStyle.Render("→ ") + line
} else if inVisual {
```

to:

```go
if isCursor {
    line = cursorStyle.Render("→ ") + cursorLineStyle.Render(line)
} else if inVisual {
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui/ -run TestDiffViewCursorLineHighlight -v`
Expected: PASS

**Step 5: Run all existing tests to verify no regressions**

Run: `go test ./internal/ui/ -v`
Expected: All tests PASS

**Step 6: Build and verify**

Run: `go build ./cmd/revui/`
Expected: Compiles without errors

**Step 7: Commit**

```bash
git add internal/ui/diffview.go internal/ui/diffview_test.go
git commit -m "feat: add subtle background highlight to cursor line"
```
