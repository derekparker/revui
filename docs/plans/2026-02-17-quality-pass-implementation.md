# Quality Pass Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Optimize, clean, and benchmark the revui codebase before pushing local changes.

**Architecture:** Conservative pass — fix concrete inefficiencies, add benchmarks for hot paths, optimize based on results, get Go expert review, and run `go fix`. No structural refactors.

**Tech Stack:** Go 1.25, lipgloss, Bubble Tea, `testing.B` for benchmarks

---

### Task 1: FileList.View() — Replace string concatenation with strings.Builder

**Files:**
- Modify: `internal/ui/filelist.go:66-84`

**Step 1: Fix the string concatenation**

In `internal/ui/filelist.go`, replace the `View()` method body. Change:

```go
func (fl FileList) View() string {
	if len(fl.files) == 0 {
		return "No changed files"
	}

	var s string
	for i, f := range fl.files {
		icon := statusIcon(f.Status)
		line := icon + " " + f.Path

		if i == fl.cursor {
			line = selectedStyle.Render("▸ " + line)
		} else {
			line = unselectedStyle.Render("  " + line)
		}
		s += line + "\n"
	}
	return s
}
```

To:

```go
func (fl FileList) View() string {
	if len(fl.files) == 0 {
		return "No changed files"
	}

	var b strings.Builder
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
	return b.String()
}
```

Add `"strings"` to the import block.

**Step 2: Run tests**

Run: `go test ./internal/ui/ -run TestFileList -v`
Expected: All FileList tests PASS

**Step 3: Commit**

```bash
git add internal/ui/filelist.go
git commit -m "refactor: use strings.Builder in FileList.View()"
```

---

### Task 2: Comment Store — Add map index for O(1) lookups

**Files:**
- Modify: `internal/comment/comment.go`

**Step 1: Update the Store struct and methods**

Replace the entire `comment.go` file content (after the `Comment` struct, which stays the same). The key change: add a `byKey` map for O(1) lookups by file+line, while keeping the `comments` slice for ordered iteration.

The key type is `commentKey{filePath, startLine}`.

```go
package comment

import "github.com/deparker/revui/internal/git"

// Comment represents an inline review comment on a diff.
type Comment struct {
	FilePath    string
	StartLine   int
	EndLine     int
	LineType    git.LineType
	Body        string
	CodeSnippet string
}

type commentKey struct {
	filePath  string
	startLine int
}

// Store holds comments in memory.
type Store struct {
	comments []Comment
	byKey    map[commentKey]int // maps key to index in comments slice
}

func NewStore() *Store {
	return &Store{
		byKey: make(map[commentKey]int),
	}
}

func (s *Store) Add(c Comment) {
	key := commentKey{c.FilePath, c.StartLine}
	if idx, ok := s.byKey[key]; ok {
		s.comments[idx] = c
		return
	}
	s.byKey[key] = len(s.comments)
	s.comments = append(s.comments, c)
}

func (s *Store) Delete(filePath string, startLine int) {
	key := commentKey{filePath, startLine}
	idx, ok := s.byKey[key]
	if !ok {
		return
	}
	last := len(s.comments) - 1
	if idx != last {
		s.comments[idx] = s.comments[last]
		movedKey := commentKey{s.comments[idx].FilePath, s.comments[idx].StartLine}
		s.byKey[movedKey] = idx
	}
	s.comments = s.comments[:last]
	delete(s.byKey, key)
}

func (s *Store) Get(filePath string, line int) *Comment {
	key := commentKey{filePath, line}
	idx, ok := s.byKey[key]
	if !ok {
		return nil
	}
	return &s.comments[idx]
}

func (s *Store) All() []Comment {
	return s.comments
}

func (s *Store) ForFile(filePath string) []Comment {
	var result []Comment
	for _, c := range s.comments {
		if c.FilePath == filePath {
			result = append(result, c)
		}
	}
	return result
}

func (s *Store) HasComment(filePath string, line int) bool {
	_, ok := s.byKey[commentKey{filePath, line}]
	return ok
}
```

Note: `Delete` uses swap-with-last to avoid slice shifting. `Add` on an existing key updates in-place. `HasComment` avoids the `Get` indirection.

**Step 2: Run tests**

Run: `go test ./internal/comment/ -v`
Expected: All 5 comment tests PASS

**Step 3: Commit**

```bash
git add internal/comment/comment.go
git commit -m "refactor: use map index for O(1) comment lookups"
```

---

### Task 3: Add benchmarks for hot paths

**Files:**
- Modify: `internal/git/parse_test.go`
- Modify: `internal/ui/diffview_test.go`
- Modify: `internal/ui/sidebyside_test.go`
- Modify: `internal/comment/format_test.go`

**Step 1: Add BenchmarkParseDiff to `internal/git/parse_test.go`**

Append to the file:

```go
func BenchmarkParseDiff(b *testing.B) {
	raw, err := os.ReadFile("testdata/simple.diff")
	if err != nil {
		b.Fatalf("reading test fixture: %v", err)
	}
	input := string(raw)
	b.ResetTimer()
	for b.Loop() {
		ParseDiff(input)
	}
}
```

**Step 2: Add benchmarks to `internal/ui/diffview_test.go`**

Append to the file:

```go
func BenchmarkRenderCodeLine(b *testing.B) {
	dv := NewDiffViewer(120, 40)
	dv.SetDiff(makeTestDiff())
	dl := dv.lines[1] // first code line (context)
	b.ResetTimer()
	for b.Loop() {
		dv.renderCodeLine(dl, 1, false)
	}
}

func BenchmarkRenderSideBySideLine(b *testing.B) {
	dv := NewDiffViewer(120, 40)
	dv.SetDiff(makeTestDiff())
	dl := dv.lines[1]
	b.ResetTimer()
	for b.Loop() {
		dv.renderSideBySideLine(dl, 1, false)
	}
}

func BenchmarkFlattenLines(b *testing.B) {
	dv := NewDiffViewer(120, 40)
	dv.diff = makeTestDiff()
	b.ResetTimer()
	for b.Loop() {
		dv.flattenLines()
	}
}

func BenchmarkView(b *testing.B) {
	dv := NewDiffViewer(120, 40)
	dv.SetDiff(makeTestDiff())
	b.ResetTimer()
	for b.Loop() {
		dv.View()
	}
}
```

**Step 3: Add BenchmarkBuildSideBySidePairs to `internal/ui/sidebyside_test.go`**

Append to the file:

```go
func BenchmarkBuildSideBySidePairs(b *testing.B) {
	lines := []git.Line{
		{Content: "context1", Type: git.LineContext, OldLineNo: 1, NewLineNo: 1},
		{Content: "old line", Type: git.LineRemoved, OldLineNo: 2},
		{Content: "new line", Type: git.LineAdded, NewLineNo: 2},
		{Content: "another new", Type: git.LineAdded, NewLineNo: 3},
		{Content: "context2", Type: git.LineContext, OldLineNo: 3, NewLineNo: 4},
	}
	b.ResetTimer()
	for b.Loop() {
		BuildSideBySidePairs(lines)
	}
}
```

**Step 4: Add BenchmarkFormat to `internal/comment/format_test.go`**

Append to the file:

```go
func BenchmarkFormat(b *testing.B) {
	comments := []Comment{
		{FilePath: "a.go", StartLine: 1, EndLine: 1, LineType: git.LineAdded, Body: "first comment", CodeSnippet: "func foo() {}"},
		{FilePath: "a.go", StartLine: 10, EndLine: 15, LineType: git.LineRemoved, Body: "second comment", CodeSnippet: "old code"},
		{FilePath: "b.go", StartLine: 5, EndLine: 5, LineType: git.LineContext, Body: "third comment", CodeSnippet: "some code"},
		{FilePath: "c.go", StartLine: 20, EndLine: 20, LineType: git.LineAdded, Body: "fourth comment", CodeSnippet: "new code"},
	}
	b.ResetTimer()
	for b.Loop() {
		Format(comments)
	}
}
```

**Step 5: Run all benchmarks**

Run: `go test ./... -bench=. -benchmem -count=3`
Expected: All benchmarks run, showing ns/op, B/op, allocs/op

**Step 6: Run all tests to verify no regressions**

Run: `go test ./... -v`
Expected: All tests PASS

**Step 7: Commit**

```bash
git add internal/git/parse_test.go internal/ui/diffview_test.go internal/ui/sidebyside_test.go internal/comment/format_test.go
git commit -m "test: add benchmarks for hot paths"
```

---

### Task 4: Optimize based on benchmark results

**Files:**
- Potentially: `internal/git/parse.go`, `internal/ui/diffview.go`, `internal/comment/format.go`

**Step 1: Run benchmarks with allocation profiling**

Run: `go test ./... -bench=. -benchmem -count=5`

**Step 2: Identify optimization targets**

Look for functions with high allocs/op or B/op relative to their work. Common wins:
- Pre-allocate slices with `make([]T, 0, capacity)` where the size is known or estimable
- Avoid `strings.Split` when you can iterate with `strings.Index` instead
- Reuse buffers where possible

**Step 3: Apply optimizations**

Apply the specific optimizations identified. Re-run benchmarks after each change to verify improvement.

**Step 4: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add -u
git commit -m "perf: reduce allocations in hot paths"
```

---

### Task 5: Go expert review

**Step 1: Dispatch Go expert agent**

Use the `go-expert` Task agent to review the full codebase at `/home/deparker/Code/revui`. The agent should:
- Review all `.go` files for idiomatic Go patterns
- Check for any non-idiomatic code, missed optimizations, or subtle bugs
- Verify error handling is correct
- Check that exported APIs are well-designed
- Note any issues that should be fixed

**Step 2: Apply fixes from the review**

Fix any issues identified by the Go expert. Run tests after each fix.

**Step 3: Commit**

```bash
git add -u
git commit -m "refactor: apply Go expert review feedback"
```

---

### Task 6: Final pass — go fix, go vet, verify

**Step 1: Run go fix**

Run: `go fix ./...`

**Step 2: Run go vet**

Run: `go vet ./...`
Expected: No issues

**Step 3: Run all tests and benchmarks**

Run: `go test ./... -v && go test ./... -bench=. -benchmem`
Expected: All tests PASS, all benchmarks run

**Step 4: Commit if any changes from go fix**

```bash
git add -u
git commit -m "chore: apply go fix"
```
