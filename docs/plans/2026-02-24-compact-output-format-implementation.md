# Compact Output Format Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce token count of review output by ~65% by switching to a compact format and removing code snippet storage.

**Architecture:** Rewrite `Format()` to produce minimal grouped output (file header + bullet comments). Remove `CodeSnippet` field from `Comment` struct and all snippet capture/passing code across UI and comment packages.

**Tech Stack:** Go, Bubble Tea TUI framework

---

### Task 1: Rewrite Format function and update tests

**Files:**
- Modify: `internal/comment/format.go:8-88` (entire file after imports)
- Modify: `internal/comment/format_test.go`

**Step 1: Update tests to expect new compact format**

Replace the test expectations in `format_test.go`. The new format uses:
- File path on its own line (no `###`)
- `- L{n}` or `- L{n}-{m}` with optional `(added)`/`(removed)` suffix
- No code snippets, no `**bold**`, no `---` separators

```go
// In TestFormatSingleComment, replace the assertion block (lines 23-37) with:
	expected := "main.go\n- L10 (added): This needs error handling.\n"
	if out != expected {
		t.Errorf("got:\n%s\nwant:\n%s", out, expected)
	}
```

```go
// In TestFormatRangeComment, replace the assertion block (lines 51-54) with:
	expected := "util.go\n- L5-8 (removed): Why was this removed?\n"
	if out != expected {
		t.Errorf("got:\n%s\nwant:\n%s", out, expected)
	}
```

```go
// In TestFormatGroupsByFile, keep existing assertions but also add a context line type test.
// The existing assertions (checking count of file headers) should still work since file paths
// still appear once each. But update them to check for "a.go\n" instead of "### a.go".
	if strings.Count(out, "a.go\n") != 1 {
		t.Error("a.go header should appear exactly once")
	}
	if strings.Count(out, "b.go\n") != 1 {
		t.Error("b.go header should appear exactly once")
	}
```

Remove `CodeSnippet` from all test comment literals — every `Comment{}` in the test file that has a `CodeSnippet` field.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/comment/ -v -run TestFormat`
Expected: FAIL — output doesn't match new expected format.

**Step 3: Rewrite Format and helpers**

Replace `Format`, `writeLineInfo`, and remove `fileExtension` in `internal/comment/format.go`:

```go
func Format(comments []Comment) string {
	if len(comments) == 0 {
		return ""
	}

	grouped := make(map[string][]Comment)
	var fileOrder []string
	seen := make(map[string]bool)
	for _, c := range comments {
		if !seen[c.FilePath] {
			fileOrder = append(fileOrder, c.FilePath)
			seen[c.FilePath] = true
		}
		grouped[c.FilePath] = append(grouped[c.FilePath], c)
	}

	var b strings.Builder
	b.Grow(64 * len(comments))

	for i, file := range fileOrder {
		b.WriteString(file)
		b.WriteByte('\n')

		for _, c := range grouped[file] {
			b.WriteString("- ")
			writeLineInfo(&b, c)
			b.WriteString(": ")
			b.WriteString(c.Body)
			b.WriteByte('\n')
		}

		if i < len(fileOrder)-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func writeLineInfo(b *strings.Builder, c Comment) {
	b.WriteByte('L')
	b.WriteString(strconv.Itoa(c.StartLine))

	if c.EndLine != 0 && c.EndLine != c.StartLine {
		b.WriteByte('-')
		b.WriteString(strconv.Itoa(c.EndLine))
	}

	lineType := c.LineType.String()
	if lineType != "context" {
		b.WriteString(" (")
		b.WriteString(lineType)
		b.WriteByte(')')
	}
}
```

Remove the `fileExtension` function entirely (lines 82-88). Remove `"strings"` from the import if it was only used by `fileExtension` — but `strings.Builder` still needs it, so keep it.

Actually `strings` is needed for `strings.Builder`. The only import to remove is nothing — keep `strconv` and `strings`.

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/comment/ -v`
Expected: PASS

**Step 5: Run benchmark to verify no regression**

Run: `go test ./internal/comment/ -bench=BenchmarkFormat -benchmem -count=3`
Expected: PASS, likely faster due to less output.

**Step 6: Commit**

```bash
git add internal/comment/format.go internal/comment/format_test.go
git commit -m "feat: compact review output format for reduced token count"
```

---

### Task 2: Remove CodeSnippet from Comment struct

**Files:**
- Modify: `internal/comment/comment.go:6-12` (Comment struct)
- Modify: `internal/comment/format_test.go` (remove CodeSnippet from test literals)

**Step 1: Remove CodeSnippet field from Comment struct**

In `internal/comment/comment.go`, remove line 12 (`CodeSnippet string`) from the `Comment` struct so it becomes:

```go
type Comment struct {
	FilePath  string
	StartLine int
	EndLine   int
	LineType  git.LineType
	Body      string
}
```

**Step 2: Remove CodeSnippet from test comment literals**

In `internal/comment/format_test.go`, remove the `CodeSnippet: ...` field from every `Comment{}` literal. There are instances at approximately lines 18, 48, 113, 114, 115, 116 (the benchmark test data).

**Step 3: Verify compilation and tests**

Run: `go build ./... && go test ./internal/comment/ -v`
Expected: Build succeeds. All tests pass.

**Step 4: Commit**

```bash
git add internal/comment/comment.go internal/comment/format_test.go
git commit -m "refactor: remove CodeSnippet field from Comment struct"
```

---

### Task 3: Remove snippet from CommentInput and CommentSubmitMsg

**Files:**
- Modify: `internal/ui/commentinput.go:17-24` (CommentSubmitMsg struct), `:29-38` (CommentInput struct), `:55` (Activate signature), `:61` (codeSnippet assignment), `:91-98` (submitMsg construction)

**Step 1: Remove CodeSnippet from CommentSubmitMsg**

In `internal/ui/commentinput.go`, remove the `CodeSnippet string` field from `CommentSubmitMsg` (line 22).

**Step 2: Remove codeSnippet from CommentInput struct**

Remove the `codeSnippet string` field (line 36) from the `CommentInput` struct.

**Step 3: Remove snippet parameter from Activate**

Change the `Activate` method signature from:
```go
func (ci *CommentInput) Activate(filePath string, lineNo, endLineNo int, lineType git.LineType, snippet, existing string) {
```
to:
```go
func (ci *CommentInput) Activate(filePath string, lineNo, endLineNo int, lineType git.LineType, existing string) {
```

Remove line 61 (`ci.codeSnippet = snippet`).

**Step 4: Remove CodeSnippet from submitMsg construction**

In the `Update` method, remove `CodeSnippet: ci.codeSnippet,` from the `CommentSubmitMsg` literal (line 96).

**Step 5: Verify the file compiles in isolation**

Run: `go vet ./internal/ui/commentinput.go`
Expected: This will fail because callers in root.go still pass the old arguments. That's expected — we fix callers in the next task.

**Step 6: Do NOT commit yet** — root.go callers still need updating.

---

### Task 4: Update root.go callers and remove SnippetRange

**Files:**
- Modify: `internal/ui/root.go:225-233` (CommentSubmitMsg handler), `:410-421` (visual mode comment), `:424-432` (single-line comment), `:434-440` (binary file comment)
- Modify: `internal/ui/diffview.go:635-645` (SnippetRange method)

**Step 1: Update CommentSubmitMsg handler in root.go**

In `root.go` around line 225-233, remove `CodeSnippet: msg.CodeSnippet,` from the `comment.Comment{}` literal.

**Step 2: Update visual mode Activate call**

Around line 410-421, remove the `snippet` variable and the `SnippetRange` call. Change:
```go
snippet := m.diffViewer.SnippetRange(vStart, vEnd)
...
m.commentInput.Activate(sel.Path, startLineNo, endLineNo, lineType, snippet, "")
```
to:
```go
m.commentInput.Activate(sel.Path, startLineNo, endLineNo, lineType, "")
```

**Step 3: Update single-line Activate call**

Around line 432, change:
```go
m.commentInput.Activate(sel.Path, lineNo, lineNo, line.Type, line.Content, existing)
```
to:
```go
m.commentInput.Activate(sel.Path, lineNo, lineNo, line.Type, existing)
```

**Step 4: Update binary file Activate call**

Around line 440, change:
```go
m.commentInput.Activate(sel.Path, 0, 0, git.LineContext, "", existing)
```
to:
```go
m.commentInput.Activate(sel.Path, 0, 0, git.LineContext, existing)
```

**Step 5: Remove SnippetRange from diffview.go**

Delete the `SnippetRange` method (lines 635-645) from `internal/ui/diffview.go`.

**Step 6: Verify everything compiles and tests pass**

Run: `go build ./... && go test ./... -v`
Expected: All packages build. All tests pass.

**Step 7: Run go vet**

Run: `go vet ./...`
Expected: No issues.

**Step 8: Commit**

```bash
git add internal/ui/commentinput.go internal/ui/root.go internal/ui/diffview.go
git commit -m "refactor: remove code snippet capture from review flow"
```
