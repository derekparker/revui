# Git-Style Diff Colors Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace Chroma syntax highlighting with standard git-style foreground coloring (green=added, red=removed, white=unchanged).

**Architecture:** Remove the `internal/syntax` package entirely. Change Lipgloss styles in `diffview.go` from background colors to foreground colors. Remove all highlighter references from the `DiffViewer` struct and rendering methods.

**Tech Stack:** Go, Lipgloss (charmbracelet), Bubble Tea

---

### Task 1: Update Lipgloss styles to foreground colors

**Files:**
- Modify: `internal/ui/diffview.go:14-17`

**Step 1: Change the style definitions**

Replace lines 14-17 of `diffview.go`:

```go
// Before:
addedLineStyle   = lipgloss.NewStyle().Background(lipgloss.Color("22"))
removedLineStyle = lipgloss.NewStyle().Background(lipgloss.Color("52"))
contextLineStyle = lipgloss.NewStyle()

// After:
addedLineStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
removedLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
contextLineStyle = lipgloss.NewStyle()
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: Success (no errors)

---

### Task 2: Remove syntax highlighting from renderCodeLine

**Files:**
- Modify: `internal/ui/diffview.go:374-387`

**Step 1: Remove highlighter call and apply styles to full line content**

Replace `renderCodeLine` lines 374-387:

```go
// Before:
text := l.Content
if dv.highlightEnabled && dv.diff != nil {
    text = dv.highlighter.HighlightLine(dv.diff.Path, l.Content)
}

var content string
switch l.Type {
case git.LineAdded:
    content = addedLineStyle.Render("+") + text
case git.LineRemoved:
    content = removedLineStyle.Render("-") + text
default:
    content = " " + text
}

// After:
var content string
switch l.Type {
case git.LineAdded:
    content = addedLineStyle.Render("+" + l.Content)
case git.LineRemoved:
    content = removedLineStyle.Render("-" + l.Content)
default:
    content = " " + l.Content
}
```

Note: The style now wraps `prefix + content` together so the entire line is colored, not just the prefix character.

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: Success

---

### Task 3: Remove syntax highlighting from renderSideBySideLine

**Files:**
- Modify: `internal/ui/diffview.go:392-444` (line numbers may have shifted after Task 2)

**Step 1: Remove highlighter call and apply styles to full line content**

Remove the highlighter block (lines 401-404):
```go
// Remove these lines:
text := l.Content
if dv.highlightEnabled && dv.diff != nil {
    text = dv.highlighter.HighlightLine(dv.diff.Path, l.Content)
}
```

Then update the three switch cases to use `l.Content` directly with foreground styles:

```go
case git.LineRemoved:
    oldNo := fmt.Sprintf("%4d ", l.OldLineNo)
    leftGutter := lineNoStyle.Render(oldNo)
    leftContent := removedLineStyle.Render("-" + l.Content)
    left := padToWidth(leftGutter+leftContent, halfWidth)
    right := padToWidth(emptyLineNo, halfWidth)
    return left + marker + sep + right

case git.LineAdded:
    left := padToWidth(emptyLineNo, halfWidth)
    newNo := fmt.Sprintf("%4d ", l.NewLineNo)
    rightGutter := lineNoStyle.Render(newNo)
    rightContent := addedLineStyle.Render("+" + l.Content)
    right := padToWidth(rightGutter+rightContent, halfWidth)
    return left + marker + sep + right

default: // context
    oldNo := fmt.Sprintf("%4d ", l.OldLineNo)
    newNo := fmt.Sprintf("%4d ", l.NewLineNo)
    leftGutter := lineNoStyle.Render(oldNo)
    left := padToWidth(leftGutter+" "+l.Content, halfWidth)
    rightGutter := lineNoStyle.Render(newNo)
    right := padToWidth(rightGutter+" "+l.Content, halfWidth)
    return left + marker + sep + right
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: Success

---

### Task 4: Remove highlighter fields and methods from DiffViewer

**Files:**
- Modify: `internal/ui/diffview.go`

**Step 1: Remove struct fields**

Remove these two fields from the `DiffViewer` struct (lines 49-50):
```go
highlighter        *syntax.Highlighter
highlightEnabled   bool
```

**Step 2: Remove from constructor**

Remove these two lines from `NewDiffViewer` (lines 67-68):
```go
highlighter:      syntax.NewHighlighter(),
highlightEnabled: true,
```

**Step 3: Remove EnableSyntaxHighlighting method**

Delete the entire method (lines 72-75):
```go
// EnableSyntaxHighlighting enables or disables syntax highlighting.
func (dv *DiffViewer) EnableSyntaxHighlighting(enabled bool) {
    dv.highlightEnabled = enabled
}
```

**Step 4: Remove the syntax import**

Remove `"github.com/deparker/revui/internal/syntax"` from the import block (line 11).

**Step 5: Verify it compiles**

Run: `go build ./...`
Expected: May fail if tests reference `EnableSyntaxHighlighting` — that's fixed in Task 5.

---

### Task 5: Update tests

**Files:**
- Modify: `internal/ui/diffview_test.go`

**Step 1: Remove the highlighting-specific test**

Delete `TestDiffViewWithHighlighting` entirely (lines 96-105):
```go
func TestDiffViewWithHighlighting(t *testing.T) {
    dv := NewDiffViewer(80, 20)
    dv.EnableSyntaxHighlighting(true)
    dv.SetDiff(makeTestDiff())

    view := dv.View()
    if view == "" {
        t.Error("expected non-empty view with highlighting")
    }
}
```

**Step 2: Update comment in TestDiffViewRenderNotEmpty**

Update the comment on lines 81-82 of `diffview_test.go`:
```go
// Before:
// Content may include ANSI escape codes from syntax highlighting,
// so check for a substring that survives highlighting

// After:
// Check that the diff content appears in the rendered view
```

**Step 3: Run tests**

Run: `go test ./internal/ui/ -v`
Expected: All tests pass

---

### Task 6: Delete syntax package and clean up dependencies

**Files:**
- Delete: `internal/syntax/highlight.go`
- Delete: `internal/syntax/highlight_test.go`

**Step 1: Delete the syntax package**

```bash
rm -rf internal/syntax/
```

**Step 2: Run go mod tidy to remove chroma dependency**

```bash
go mod tidy
```

**Step 3: Verify full build and all tests pass**

```bash
go build ./...
go test ./... -v
```

Expected: Clean build, all tests pass, no reference to chroma in go.mod.

**Step 4: Verify chroma is gone from go.mod**

```bash
grep chroma go.mod
```

Expected: No output (chroma fully removed).

---

### Task 7: Commit

**Step 1: Stage and commit all changes**

```bash
git add -A
git commit -m "feat: replace syntax highlighting with git-style diff colors

Remove Chroma-based syntax highlighting in favor of standard
git-style foreground colors: green for added lines, red for
removed lines, and default terminal color for context lines."
```
