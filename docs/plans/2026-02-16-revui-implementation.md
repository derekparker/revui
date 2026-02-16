# revui Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a TUI code review tool that shows branch diffs with vim-style navigation, inline commenting, and clipboard output for AI agents.

**Architecture:** Composable Bubble Tea sub-models (FileList, DiffViewer, CommentInput) orchestrated by a RootModel. Git integration via os/exec. Syntax highlighting via Chroma. Comments stored in memory, formatted as markdown and copied to clipboard on finish.

**Tech Stack:** Go, charmbracelet/bubbletea, charmbracelet/lipgloss, charmbracelet/bubbles, alecthomas/chroma/v2, atotto/clipboard

**Design doc:** `docs/plans/2026-02-16-revui-design.md`

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/revui/main.go`

**Step 1: Initialize Go module**

```bash
cd /home/deparker/Code/revui
go mod init github.com/deparker/revui
```

**Step 2: Create minimal main.go**

Create `cmd/revui/main.go`:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("revui")
	os.Exit(0)
}
```

**Step 3: Verify it compiles and runs**

```bash
go run ./cmd/revui
```

Expected: prints `revui`

**Step 4: Commit**

```bash
git add go.mod cmd/
git commit -m "feat: initialize project with go module and main entry point"
```

---

## Task 2: Git Data Models

**Files:**
- Create: `internal/git/types.go`
- Create: `internal/git/types_test.go`

**Step 1: Write tests for data model types**

Create `internal/git/types_test.go`:

```go
package git

import "testing"

func TestLineTypeString(t *testing.T) {
	tests := []struct {
		lt   LineType
		want string
	}{
		{LineAdded, "added"},
		{LineRemoved, "removed"},
		{LineContext, "context"},
	}
	for _, tt := range tests {
		if got := tt.lt.String(); got != tt.want {
			t.Errorf("LineType(%d).String() = %q, want %q", tt.lt, got, tt.want)
		}
	}
}

func TestFileStatusString(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"A", "added"},
		{"M", "modified"},
		{"D", "deleted"},
		{"R", "renamed"},
		{"X", "unknown"},
	}
	for _, tt := range tests {
		if got := FileStatusString(tt.status); got != tt.want {
			t.Errorf("FileStatusString(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/git/ -v
```

Expected: FAIL — types not defined

**Step 3: Implement types**

Create `internal/git/types.go`:

```go
package git

// LineType represents the type of a diff line.
type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
)

func (lt LineType) String() string {
	switch lt {
	case LineAdded:
		return "added"
	case LineRemoved:
		return "removed"
	default:
		return "context"
	}
}

// FileStatusString returns a human-readable string for a git status code.
func FileStatusString(status string) string {
	switch status {
	case "A":
		return "added"
	case "M":
		return "modified"
	case "D":
		return "deleted"
	case "R":
		return "renamed"
	default:
		return "unknown"
	}
}

// Line represents a single line in a diff.
type Line struct {
	Content   string
	Type      LineType
	OldLineNo int
	NewLineNo int
}

// Hunk represents a contiguous section of a diff.
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Header   string
	Lines    []Line
}

// FileDiff represents the diff for a single file.
type FileDiff struct {
	Path   string
	Status string // A, M, D, R
	Hunks  []Hunk
}

// ChangedFile represents a file that changed between two refs.
type ChangedFile struct {
	Path   string
	Status string
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/git/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/git/
git commit -m "feat: add git diff data model types"
```

---

## Task 3: Diff Parser

**Files:**
- Create: `internal/git/parse.go`
- Create: `internal/git/parse_test.go`
- Create: `internal/git/testdata/simple.diff`

**Step 1: Create a test fixture diff**

Create `internal/git/testdata/simple.diff`:

```
diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main

 func main() {
-	fmt.Println("hello")
+	fmt.Println("hello world")
+	fmt.Println("goodbye")
 }
```

**Step 2: Write parser tests**

Create `internal/git/parse_test.go`:

```go
package git

import (
	"os"
	"testing"
)

func TestParseFileDiff(t *testing.T) {
	data, err := os.ReadFile("testdata/simple.diff")
	if err != nil {
		t.Fatal(err)
	}

	diffs, err := ParseDiff(string(data))
	if err != nil {
		t.Fatal(err)
	}

	if len(diffs) != 1 {
		t.Fatalf("expected 1 file diff, got %d", len(diffs))
	}

	fd := diffs[0]
	if fd.Path != "main.go" {
		t.Errorf("path = %q, want %q", fd.Path, "main.go")
	}

	if len(fd.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(fd.Hunks))
	}

	h := fd.Hunks[0]
	if h.OldStart != 1 || h.OldCount != 5 {
		t.Errorf("old range = %d,%d, want 1,5", h.OldStart, h.OldCount)
	}
	if h.NewStart != 1 || h.NewCount != 6 {
		t.Errorf("new range = %d,%d, want 1,6", h.NewStart, h.NewCount)
	}

	// Verify line types
	var added, removed, context int
	for _, l := range h.Lines {
		switch l.Type {
		case LineAdded:
			added++
		case LineRemoved:
			removed++
		case LineContext:
			context++
		}
	}
	if added != 2 {
		t.Errorf("added lines = %d, want 2", added)
	}
	if removed != 1 {
		t.Errorf("removed lines = %d, want 1", removed)
	}
	if context != 3 {
		t.Errorf("context lines = %d, want 3", context)
	}
}

func TestParseLineNumbers(t *testing.T) {
	data, err := os.ReadFile("testdata/simple.diff")
	if err != nil {
		t.Fatal(err)
	}

	diffs, err := ParseDiff(string(data))
	if err != nil {
		t.Fatal(err)
	}

	lines := diffs[0].Hunks[0].Lines
	// First line: context "package main" → old=1, new=1
	if lines[0].OldLineNo != 1 || lines[0].NewLineNo != 1 {
		t.Errorf("line 0: old=%d new=%d, want old=1 new=1",
			lines[0].OldLineNo, lines[0].NewLineNo)
	}

	// Removed line should have old line number, no new
	for _, l := range lines {
		if l.Type == LineRemoved {
			if l.OldLineNo == 0 {
				t.Error("removed line should have an old line number")
			}
			if l.NewLineNo != 0 {
				t.Error("removed line should not have a new line number")
			}
		}
	}

	// Added lines should have new line number, no old
	for _, l := range lines {
		if l.Type == LineAdded {
			if l.NewLineNo == 0 {
				t.Error("added line should have a new line number")
			}
			if l.OldLineNo != 0 {
				t.Error("added line should not have an old line number")
			}
		}
	}
}
```

**Step 3: Run tests to verify they fail**

```bash
go test ./internal/git/ -v -run TestParse
```

Expected: FAIL — ParseDiff not defined

**Step 4: Implement the diff parser**

Create `internal/git/parse.go`:

```go
package git

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// ParseDiff parses a unified diff string into a slice of FileDiff.
func ParseDiff(raw string) ([]FileDiff, error) {
	var diffs []FileDiff
	lines := strings.Split(raw, "\n")

	var current *FileDiff
	var currentHunk *Hunk
	var oldLine, newLine int

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// New file diff starts with "diff --git"
		if strings.HasPrefix(line, "diff --git") {
			if current != nil {
				if currentHunk != nil {
					current.Hunks = append(current.Hunks, *currentHunk)
					currentHunk = nil
				}
				diffs = append(diffs, *current)
			}
			current = &FileDiff{}
			currentHunk = nil
			continue
		}

		if current == nil {
			continue
		}

		// Parse file path from +++ line
		if strings.HasPrefix(line, "+++ b/") {
			current.Path = strings.TrimPrefix(line, "+++ b/")
			continue
		}
		if strings.HasPrefix(line, "+++ /dev/null") {
			// File was deleted — path comes from --- line
			continue
		}
		if strings.HasPrefix(line, "--- a/") {
			if current.Path == "" {
				current.Path = strings.TrimPrefix(line, "--- a/")
			}
			continue
		}

		// Skip other header lines (index, old mode, new mode, etc.)
		if strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "old mode") ||
			strings.HasPrefix(line, "new mode") ||
			strings.HasPrefix(line, "new file mode") ||
			strings.HasPrefix(line, "deleted file mode") ||
			strings.HasPrefix(line, "similarity index") ||
			strings.HasPrefix(line, "rename from") ||
			strings.HasPrefix(line, "rename to") ||
			strings.HasPrefix(line, "Binary files") ||
			strings.HasPrefix(line, "--- /dev/null") {
			continue
		}

		// Hunk header
		if m := hunkHeaderRe.FindStringSubmatch(line); m != nil {
			if currentHunk != nil {
				current.Hunks = append(current.Hunks, *currentHunk)
			}

			oldStart, _ := strconv.Atoi(m[1])
			oldCount := 1
			if m[2] != "" {
				oldCount, _ = strconv.Atoi(m[2])
			}
			newStart, _ := strconv.Atoi(m[3])
			newCount := 1
			if m[4] != "" {
				newCount, _ = strconv.Atoi(m[4])
			}

			currentHunk = &Hunk{
				OldStart: oldStart,
				OldCount: oldCount,
				NewStart: newStart,
				NewCount: newCount,
				Header:   line,
			}
			oldLine = oldStart
			newLine = newStart
			continue
		}

		if currentHunk == nil {
			continue
		}

		// Diff content lines
		if strings.HasPrefix(line, "+") {
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Content:   strings.TrimPrefix(line, "+"),
				Type:      LineAdded,
				NewLineNo: newLine,
			})
			newLine++
		} else if strings.HasPrefix(line, "-") {
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Content:   strings.TrimPrefix(line, "-"),
				Type:      LineRemoved,
				OldLineNo: oldLine,
			})
			oldLine++
		} else if strings.HasPrefix(line, " ") || line == "" {
			// Context line (or empty context line at end)
			content := line
			if strings.HasPrefix(line, " ") {
				content = line[1:]
			}
			// Skip trailing empty lines after last hunk
			if line == "" && i == len(lines)-1 {
				continue
			}
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Content:   content,
				Type:      LineContext,
				OldLineNo: oldLine,
				NewLineNo: newLine,
			})
			oldLine++
			newLine++
		} else if strings.HasPrefix(line, `\ No newline at end of file`) {
			continue
		} else {
			return nil, fmt.Errorf("unexpected line in diff at line %d: %q", i+1, line)
		}
	}

	// Flush last file/hunk
	if current != nil {
		if currentHunk != nil {
			current.Hunks = append(current.Hunks, *currentHunk)
		}
		diffs = append(diffs, *current)
	}

	return diffs, nil
}

// ParseNameStatus parses the output of `git diff --name-status` into ChangedFile entries.
func ParseNameStatus(raw string) []ChangedFile {
	var files []ChangedFile
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		files = append(files, ChangedFile{
			Status: parts[0],
			Path:   parts[1],
		})
	}
	return files
}
```

**Step 5: Run tests to verify they pass**

```bash
go test ./internal/git/ -v
```

Expected: PASS

**Step 6: Commit**

```bash
git add internal/git/
git commit -m "feat: add unified diff parser with line number tracking"
```

---

## Task 4: Git Command Runner

**Files:**
- Create: `internal/git/git.go`
- Create: `internal/git/git_test.go`

This wraps os/exec calls to git. Tests use a real temporary git repo.

**Step 1: Write tests**

Create `internal/git/git_test.go`:

```go
package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temp git repo with a commit on main and a feature branch.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "checkout", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", args, err, out)
		}
	}

	// Create initial file and commit
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n\nfunc hello() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "initial"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", args, err, out)
		}
	}

	// Create feature branch with changes
	for _, args := range [][]string{
		{"git", "checkout", "-b", "feature"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", args, err, out)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n\nfunc hello() {\n\tfmt.Println(\"hello\")\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "world.go"), []byte("package main\n\nfunc world() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "add feature"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", args, err, out)
		}
	}

	return dir
}

func TestChangedFiles(t *testing.T) {
	dir := setupTestRepo(t)

	r := &Runner{Dir: dir}
	files, err := r.ChangedFiles("main")
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 changed files, got %d", len(files))
	}

	paths := map[string]string{}
	for _, f := range files {
		paths[f.Path] = f.Status
	}

	if paths["hello.go"] != "M" {
		t.Errorf("hello.go status = %q, want M", paths["hello.go"])
	}
	if paths["world.go"] != "A" {
		t.Errorf("world.go status = %q, want A", paths["world.go"])
	}
}

func TestFileDiff(t *testing.T) {
	dir := setupTestRepo(t)

	r := &Runner{Dir: dir}
	fd, err := r.FileDiff("main", "hello.go")
	if err != nil {
		t.Fatal(err)
	}

	if fd.Path != "hello.go" {
		t.Errorf("path = %q, want %q", fd.Path, "hello.go")
	}
	if len(fd.Hunks) == 0 {
		t.Fatal("expected at least one hunk")
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := setupTestRepo(t)

	r := &Runner{Dir: dir}
	branch, err := r.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feature" {
		t.Errorf("branch = %q, want %q", branch, "feature")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/git/ -v -run "TestChanged|TestFileDiff|TestCurrentBranch"
```

Expected: FAIL — Runner not defined

**Step 3: Implement git command runner**

Create `internal/git/git.go`:

```go
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes git commands in a directory.
type Runner struct {
	Dir string
}

// CurrentBranch returns the name of the current git branch.
func (r *Runner) CurrentBranch() (string, error) {
	out, err := r.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// ChangedFiles returns the list of files changed between base and HEAD.
func (r *Runner) ChangedFiles(base string) ([]ChangedFile, error) {
	out, err := r.run("diff", "--name-status", base+"..HEAD")
	if err != nil {
		return nil, fmt.Errorf("getting changed files: %w", err)
	}
	return ParseNameStatus(out), nil
}

// FileDiff returns the parsed diff for a single file between base and HEAD.
func (r *Runner) FileDiff(base, path string) (*FileDiff, error) {
	out, err := r.run("diff", base+"..HEAD", "--", path)
	if err != nil {
		return nil, fmt.Errorf("getting diff for %s: %w", path, err)
	}
	diffs, err := ParseDiff(out)
	if err != nil {
		return nil, err
	}
	if len(diffs) == 0 {
		return &FileDiff{Path: path}, nil
	}
	diffs[0].Path = path
	return &diffs[0], nil
}

// IsGitRepo checks if the directory is inside a git repository.
func (r *Runner) IsGitRepo() bool {
	_, err := r.run("rev-parse", "--git-dir")
	return err == nil
}

// BranchExists checks if a branch exists.
func (r *Runner) BranchExists(branch string) bool {
	_, err := r.run("rev-parse", "--verify", branch)
	return err == nil
}

func (r *Runner) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return "", err
	}
	return string(out), nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/git/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/git/
git commit -m "feat: add git command runner for branch diffs and file listing"
```

---

## Task 5: Comment Model and Clipboard Formatter

**Files:**
- Create: `internal/comment/comment.go`
- Create: `internal/comment/format.go`
- Create: `internal/comment/format_test.go`

**Step 1: Write formatter tests**

Create `internal/comment/format_test.go`:

```go
package comment

import (
	"strings"
	"testing"

	"github.com/deparker/revui/internal/git"
)

func TestFormatSingleComment(t *testing.T) {
	store := NewStore()
	store.Add(Comment{
		FilePath:  "main.go",
		StartLine: 10,
		EndLine:   10,
		LineType:  git.LineAdded,
		Body:      "This needs error handling.",
		CodeSnippet: "func doThing() {",
	})

	out := Format(store.All())

	if !strings.Contains(out, "## Code Review Comments") {
		t.Error("missing header")
	}
	if !strings.Contains(out, "### main.go") {
		t.Error("missing file path")
	}
	if !strings.Contains(out, "Line 10 (added)") {
		t.Error("missing line info")
	}
	if !strings.Contains(out, "This needs error handling.") {
		t.Error("missing comment body")
	}
	if !strings.Contains(out, "func doThing() {") {
		t.Error("missing code snippet")
	}
}

func TestFormatRangeComment(t *testing.T) {
	store := NewStore()
	store.Add(Comment{
		FilePath:  "util.go",
		StartLine: 5,
		EndLine:   8,
		LineType:  git.LineRemoved,
		Body:      "Why was this removed?",
		CodeSnippet: "old code here",
	})

	out := Format(store.All())
	if !strings.Contains(out, "Lines 5-8 (removed)") {
		t.Error("missing range line info")
	}
}

func TestFormatGroupsByFile(t *testing.T) {
	store := NewStore()
	store.Add(Comment{FilePath: "a.go", StartLine: 1, EndLine: 1, Body: "first"})
	store.Add(Comment{FilePath: "b.go", StartLine: 1, EndLine: 1, Body: "second"})
	store.Add(Comment{FilePath: "a.go", StartLine: 5, EndLine: 5, Body: "third"})

	out := Format(store.All())

	// a.go should appear once as a header, with two comments under it
	if strings.Count(out, "### a.go") != 1 {
		t.Error("a.go header should appear exactly once")
	}
	if strings.Count(out, "### b.go") != 1 {
		t.Error("b.go header should appear exactly once")
	}
}

func TestStoreAddAndDelete(t *testing.T) {
	store := NewStore()
	store.Add(Comment{FilePath: "a.go", StartLine: 1, EndLine: 1, Body: "hello"})
	store.Add(Comment{FilePath: "a.go", StartLine: 5, EndLine: 5, Body: "world"})

	if len(store.All()) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(store.All()))
	}

	store.Delete("a.go", 1)
	if len(store.All()) != 1 {
		t.Fatalf("expected 1 comment after delete, got %d", len(store.All()))
	}

	c := store.Get("a.go", 5)
	if c == nil || c.Body != "world" {
		t.Error("wrong comment returned after delete")
	}
}

func TestStoreGetByFileLine(t *testing.T) {
	store := NewStore()
	store.Add(Comment{FilePath: "a.go", StartLine: 10, EndLine: 10, Body: "found"})

	c := store.Get("a.go", 10)
	if c == nil {
		t.Fatal("expected comment, got nil")
	}
	if c.Body != "found" {
		t.Errorf("body = %q, want %q", c.Body, "found")
	}

	c = store.Get("a.go", 11)
	if c != nil {
		t.Error("expected nil for non-existent line")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/comment/ -v
```

Expected: FAIL — package not found

**Step 3: Implement comment model**

Create `internal/comment/comment.go`:

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

// Store holds comments in memory.
type Store struct {
	comments []Comment
}

// NewStore creates a new empty comment store.
func NewStore() *Store {
	return &Store{}
}

// Add adds a comment to the store.
func (s *Store) Add(c Comment) {
	s.comments = append(s.comments, c)
}

// Delete removes the comment at the given file and start line.
func (s *Store) Delete(filePath string, startLine int) {
	for i, c := range s.comments {
		if c.FilePath == filePath && c.StartLine == startLine {
			s.comments = append(s.comments[:i], s.comments[i+1:]...)
			return
		}
	}
}

// Get returns the comment at the given file and line, or nil if none exists.
func (s *Store) Get(filePath string, line int) *Comment {
	for i := range s.comments {
		if s.comments[i].FilePath == filePath && s.comments[i].StartLine == line {
			return &s.comments[i]
		}
	}
	return nil
}

// All returns all comments.
func (s *Store) All() []Comment {
	return s.comments
}

// ForFile returns all comments for the given file path.
func (s *Store) ForFile(filePath string) []Comment {
	var result []Comment
	for _, c := range s.comments {
		if c.FilePath == filePath {
			result = append(result, c)
		}
	}
	return result
}

// HasComment returns true if there is a comment at the given file and line.
func (s *Store) HasComment(filePath string, line int) bool {
	return s.Get(filePath, line) != nil
}
```

**Step 4: Implement formatter**

Create `internal/comment/format.go`:

```go
package comment

import (
	"fmt"
	"strings"
)

// Format renders all comments as structured markdown for clipboard output.
func Format(comments []Comment) string {
	if len(comments) == 0 {
		return ""
	}

	// Group by file
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
	b.WriteString("## Code Review Comments\n\n")

	for i, file := range fileOrder {
		b.WriteString(fmt.Sprintf("### %s\n\n", file))

		for _, c := range grouped[file] {
			// Line info
			lineInfo := formatLineInfo(c)
			b.WriteString(fmt.Sprintf("**%s:**\n", lineInfo))

			// Code snippet
			if c.CodeSnippet != "" {
				ext := fileExtension(file)
				b.WriteString(fmt.Sprintf("```%s\n%s\n```\n", ext, c.CodeSnippet))
			}

			// Comment body
			b.WriteString(fmt.Sprintf("**Comment:** %s\n\n", c.Body))
		}

		if i < len(fileOrder)-1 {
			b.WriteString("---\n\n")
		}
	}

	return b.String()
}

func formatLineInfo(c Comment) string {
	lineType := ""
	if c.LineType.String() != "context" {
		lineType = fmt.Sprintf(" (%s)", c.LineType.String())
	}
	if c.StartLine == c.EndLine || c.EndLine == 0 {
		return fmt.Sprintf("Line %d%s", c.StartLine, lineType)
	}
	return fmt.Sprintf("Lines %d-%d%s", c.StartLine, c.EndLine, lineType)
}

func fileExtension(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}
```

**Step 5: Run tests to verify they pass**

```bash
go test ./internal/comment/ -v
```

Expected: PASS

**Step 6: Commit**

```bash
git add internal/comment/
git commit -m "feat: add comment store and clipboard markdown formatter"
```

---

## Task 6: Syntax Highlighting with Chroma

**Files:**
- Create: `internal/syntax/highlight.go`
- Create: `internal/syntax/highlight_test.go`

**Step 1: Install chroma dependency**

```bash
go get github.com/alecthomas/chroma/v2
```

**Step 2: Write tests**

Create `internal/syntax/highlight_test.go`:

```go
package syntax

import "testing"

func TestHighlightGoLine(t *testing.T) {
	h := NewHighlighter()
	result := h.HighlightLine("main.go", "func hello() {")
	if result == "" {
		t.Error("expected non-empty highlighted output")
	}
	// The output should contain ANSI escape codes
	if result == "func hello() {" {
		t.Error("expected ANSI-styled output, got plain text")
	}
}

func TestHighlightUnknownExtension(t *testing.T) {
	h := NewHighlighter()
	result := h.HighlightLine("unknown.xyz", "some content")
	// Should fall back to plain text without error
	if result == "" {
		t.Error("expected non-empty output even for unknown extension")
	}
}

func TestHighlightEmptyLine(t *testing.T) {
	h := NewHighlighter()
	result := h.HighlightLine("main.go", "")
	// Empty line should return empty or whitespace
	if len(result) > 10 {
		t.Errorf("expected minimal output for empty line, got %q", result)
	}
}
```

**Step 3: Run tests to verify they fail**

```bash
go test ./internal/syntax/ -v
```

Expected: FAIL — package not defined

**Step 4: Implement highlighter**

Create `internal/syntax/highlight.go`:

```go
package syntax

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlighter provides syntax highlighting for code lines.
type Highlighter struct {
	style     *chroma.Style
	formatter *formatters.TTY256
}

// NewHighlighter creates a highlighter with a terminal-friendly dark theme.
func NewHighlighter() *Highlighter {
	return &Highlighter{
		style:     styles.Get("monokai"),
		formatter: formatters.TTY256,
	}
}

// HighlightLine applies syntax highlighting to a single line of code.
func (h *Highlighter) HighlightLine(filename, line string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, line)
	if err != nil {
		return line
	}

	var buf bytes.Buffer
	err = h.formatter.Format(&buf, h.style, iterator)
	if err != nil {
		return line
	}

	// Chroma adds a trailing newline; strip it
	return strings.TrimRight(buf.String(), "\n")
}

// ExtensionFromPath returns the file extension for lexer matching.
func ExtensionFromPath(path string) string {
	return filepath.Ext(path)
}
```

**Step 5: Run tests to verify they pass**

```bash
go test ./internal/syntax/ -v
```

Expected: PASS

**Step 6: Commit**

```bash
go get github.com/alecthomas/chroma/v2
git add internal/syntax/ go.mod go.sum
git commit -m "feat: add chroma-based syntax highlighter for diff lines"
```

---

## Task 7: File List Sub-Model

**Files:**
- Create: `internal/ui/filelist.go`
- Create: `internal/ui/filelist_test.go`

**Step 1: Install Bubble Tea dependencies**

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles
```

**Step 2: Write tests**

Create `internal/ui/filelist_test.go`:

```go
package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/deparker/revui/internal/git"
)

func TestFileListNavigation(t *testing.T) {
	files := []git.ChangedFile{
		{Path: "a.go", Status: "M"},
		{Path: "b.go", Status: "A"},
		{Path: "c.go", Status: "D"},
	}

	fl := NewFileList(files, 20, 10)

	// Initial cursor at 0
	if fl.SelectedIndex() != 0 {
		t.Errorf("initial cursor = %d, want 0", fl.SelectedIndex())
	}

	// Move down with j
	fl, _ = fl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if fl.SelectedIndex() != 1 {
		t.Errorf("after j: cursor = %d, want 1", fl.SelectedIndex())
	}

	// Move up with k
	fl, _ = fl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if fl.SelectedIndex() != 0 {
		t.Errorf("after k: cursor = %d, want 0", fl.SelectedIndex())
	}

	// k at top stays at 0
	fl, _ = fl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if fl.SelectedIndex() != 0 {
		t.Errorf("k at top: cursor = %d, want 0", fl.SelectedIndex())
	}
}

func TestFileListSelectedFile(t *testing.T) {
	files := []git.ChangedFile{
		{Path: "a.go", Status: "M"},
		{Path: "b.go", Status: "A"},
	}

	fl := NewFileList(files, 20, 10)
	if fl.SelectedFile().Path != "a.go" {
		t.Errorf("selected = %q, want %q", fl.SelectedFile().Path, "a.go")
	}

	fl, _ = fl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if fl.SelectedFile().Path != "b.go" {
		t.Errorf("selected = %q, want %q", fl.SelectedFile().Path, "b.go")
	}
}

func TestFileListGAndGG(t *testing.T) {
	files := []git.ChangedFile{
		{Path: "a.go", Status: "M"},
		{Path: "b.go", Status: "A"},
		{Path: "c.go", Status: "D"},
	}

	fl := NewFileList(files, 20, 10)

	// G jumps to bottom
	fl, _ = fl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if fl.SelectedIndex() != 2 {
		t.Errorf("after G: cursor = %d, want 2", fl.SelectedIndex())
	}
}

func TestFileListViewNotEmpty(t *testing.T) {
	files := []git.ChangedFile{
		{Path: "main.go", Status: "M"},
	}
	fl := NewFileList(files, 20, 10)
	view := fl.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}
```

**Step 3: Run tests to verify they fail**

```bash
go test ./internal/ui/ -v
```

Expected: FAIL — NewFileList not defined

**Step 4: Implement file list sub-model**

Create `internal/ui/filelist.go`:

```go
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/git"
)

var (
	selectedStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	unselectedStyle = lipgloss.NewStyle()
	statusAddedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	statusModifiedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	statusDeletedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

// FileList is a Bubble Tea sub-model for displaying changed files.
type FileList struct {
	files    []git.ChangedFile
	cursor   int
	width    int
	height   int
	focused  bool
}

// NewFileList creates a new file list with the given changed files.
func NewFileList(files []git.ChangedFile, width, height int) FileList {
	return FileList{
		files:  files,
		cursor: 0,
		width:  width,
		height: height,
	}
}

// Init returns no initial command.
func (fl FileList) Init() tea.Cmd {
	return nil
}

// Update handles key messages for vim-style navigation.
func (fl FileList) Update(msg tea.Msg) (FileList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if fl.cursor < len(fl.files)-1 {
				fl.cursor++
			}
		case "k", "up":
			if fl.cursor > 0 {
				fl.cursor--
			}
		case "G":
			fl.cursor = len(fl.files) - 1
		case "g":
			// gg handled by root model tracking "g" prefix
			fl.cursor = 0
		}
	}
	return fl, nil
}

// View renders the file list.
func (fl FileList) View() string {
	if len(fl.files) == 0 {
		return "No changed files"
	}

	var s string
	for i, f := range fl.files {
		statusIcon := statusIcon(f.Status)
		line := statusIcon + " " + f.Path

		if i == fl.cursor {
			line = selectedStyle.Render("▸ " + line)
		} else {
			line = unselectedStyle.Render("  " + line)
		}
		s += line + "\n"
	}
	return s
}

// SelectedFile returns the currently selected file.
func (fl FileList) SelectedFile() git.ChangedFile {
	if fl.cursor >= 0 && fl.cursor < len(fl.files) {
		return fl.files[fl.cursor]
	}
	return git.ChangedFile{}
}

// SelectedIndex returns the current cursor index.
func (fl FileList) SelectedIndex() int {
	return fl.cursor
}

// SetFocused sets whether this component has focus.
func (fl *FileList) SetFocused(focused bool) {
	fl.focused = focused
}

// SetSize updates the dimensions.
func (fl *FileList) SetSize(width, height int) {
	fl.width = width
	fl.height = height
}

func statusIcon(status string) string {
	switch status {
	case "A":
		return statusAddedStyle.Render("A")
	case "M":
		return statusModifiedStyle.Render("M")
	case "D":
		return statusDeletedStyle.Render("D")
	case "R":
		return statusModifiedStyle.Render("R")
	default:
		return "?"
	}
}
```

**Step 5: Run tests to verify they pass**

```bash
go test ./internal/ui/ -v
```

Expected: PASS

**Step 6: Commit**

```bash
git add internal/ui/ go.mod go.sum
git commit -m "feat: add file list sub-model with vim navigation"
```

---

## Task 8: Diff Viewer Sub-Model (Unified View)

**Files:**
- Create: `internal/ui/diffview.go`
- Create: `internal/ui/diffview_test.go`

**Step 1: Write tests**

Create `internal/ui/diffview_test.go`:

```go
package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/deparker/revui/internal/git"
)

func makeTestDiff() *git.FileDiff {
	return &git.FileDiff{
		Path:   "test.go",
		Status: "M",
		Hunks: []git.Hunk{
			{
				Header:   "@@ -1,3 +1,4 @@",
				OldStart: 1, OldCount: 3,
				NewStart: 1, NewCount: 4,
				Lines: []git.Line{
					{Content: "package main", Type: git.LineContext, OldLineNo: 1, NewLineNo: 1},
					{Content: "old line", Type: git.LineRemoved, OldLineNo: 2},
					{Content: "new line", Type: git.LineAdded, NewLineNo: 2},
					{Content: "another new", Type: git.LineAdded, NewLineNo: 3},
					{Content: "unchanged", Type: git.LineContext, OldLineNo: 3, NewLineNo: 4},
				},
			},
		},
	}
}

func TestDiffViewNavigation(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())

	if dv.CursorLine() != 0 {
		t.Errorf("initial cursor = %d, want 0", dv.CursorLine())
	}

	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if dv.CursorLine() != 1 {
		t.Errorf("after j: cursor = %d, want 1", dv.CursorLine())
	}

	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if dv.CursorLine() != 0 {
		t.Errorf("after k: cursor = %d, want 0", dv.CursorLine())
	}
}

func TestDiffViewCurrentLine(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())

	line := dv.CurrentLine()
	if line == nil {
		t.Fatal("expected non-nil line")
	}
	if line.Content != "package main" {
		t.Errorf("content = %q, want %q", line.Content, "package main")
	}
}

func TestDiffViewRenderNotEmpty(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())

	view := dv.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "package main") {
		t.Error("expected view to contain diff content")
	}
}

func TestDiffViewNoDiff(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	view := dv.View()
	if view == "" {
		t.Error("expected non-empty view even with no diff")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/ -v -run TestDiffView
```

Expected: FAIL — NewDiffViewer not defined

**Step 3: Implement diff viewer**

Create `internal/ui/diffview.go`:

```go
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/git"
)

var (
	addedLineStyle   = lipgloss.NewStyle().Background(lipgloss.Color("22"))
	removedLineStyle = lipgloss.NewStyle().Background(lipgloss.Color("52"))
	contextLineStyle = lipgloss.NewStyle()
	hunkHeaderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Faint(true)
	lineNoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(6)
	cursorStyle      = lipgloss.NewStyle().Bold(true)
	commentMarker    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("●")
)

// diffLine is a flattened line for display, which can be a hunk header or a code line.
type diffLine struct {
	isHunkHeader bool
	hunkHeader   string
	line         *git.Line
	fileLineIdx  int // index into the flattened lines slice
}

// DiffViewer is a Bubble Tea sub-model for displaying file diffs.
type DiffViewer struct {
	diff         *git.FileDiff
	lines        []diffLine
	cursor       int
	offset       int // scroll offset
	width        int
	height       int
	focused      bool
	commentLines map[int]bool // lines with comments (by flattened index)
}

// NewDiffViewer creates a new diff viewer.
func NewDiffViewer(width, height int) DiffViewer {
	return DiffViewer{
		width:        width,
		height:       height,
		commentLines: make(map[int]bool),
	}
}

// SetDiff sets the diff content to display.
func (dv *DiffViewer) SetDiff(fd *git.FileDiff) {
	dv.diff = fd
	dv.cursor = 0
	dv.offset = 0
	dv.lines = dv.flattenLines()
}

// SetCommentLines updates which lines have comments.
func (dv *DiffViewer) SetCommentLines(lines map[int]bool) {
	dv.commentLines = lines
}

func (dv *DiffViewer) flattenLines() []diffLine {
	if dv.diff == nil {
		return nil
	}
	var result []diffLine
	for _, h := range dv.diff.Hunks {
		result = append(result, diffLine{
			isHunkHeader: true,
			hunkHeader:   h.Header,
		})
		for i := range h.Lines {
			result = append(result, diffLine{
				line: &h.Lines[i],
			})
		}
	}
	return result
}

// Init returns no initial command.
func (dv DiffViewer) Init() tea.Cmd {
	return nil
}

// Update handles key messages for vim-style navigation.
func (dv DiffViewer) Update(msg tea.Msg) (DiffViewer, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if dv.cursor < len(dv.lines)-1 {
				dv.cursor++
				dv.adjustScroll()
			}
		case "k", "up":
			if dv.cursor > 0 {
				dv.cursor--
				dv.adjustScroll()
			}
		case "G":
			dv.cursor = len(dv.lines) - 1
			dv.adjustScroll()
		case "g":
			dv.cursor = 0
			dv.offset = 0
		case "ctrl+d":
			dv.cursor += dv.height / 2
			if dv.cursor >= len(dv.lines) {
				dv.cursor = len(dv.lines) - 1
			}
			dv.adjustScroll()
		case "ctrl+u":
			dv.cursor -= dv.height / 2
			if dv.cursor < 0 {
				dv.cursor = 0
			}
			dv.adjustScroll()
		case "ctrl+f":
			dv.cursor += dv.height
			if dv.cursor >= len(dv.lines) {
				dv.cursor = len(dv.lines) - 1
			}
			dv.adjustScroll()
		case "ctrl+b":
			dv.cursor -= dv.height
			if dv.cursor < 0 {
				dv.cursor = 0
			}
			dv.adjustScroll()
		case "}":
			dv.jumpToNextHunk()
		case "{":
			dv.jumpToPrevHunk()
		}
	}
	return dv, nil
}

func (dv *DiffViewer) adjustScroll() {
	if dv.cursor < dv.offset {
		dv.offset = dv.cursor
	}
	if dv.cursor >= dv.offset+dv.height {
		dv.offset = dv.cursor - dv.height + 1
	}
}

func (dv *DiffViewer) jumpToNextHunk() {
	for i := dv.cursor + 1; i < len(dv.lines); i++ {
		if dv.lines[i].isHunkHeader {
			dv.cursor = i
			dv.adjustScroll()
			return
		}
	}
}

func (dv *DiffViewer) jumpToPrevHunk() {
	for i := dv.cursor - 1; i >= 0; i-- {
		if dv.lines[i].isHunkHeader {
			dv.cursor = i
			dv.adjustScroll()
			return
		}
	}
}

// View renders the diff.
func (dv DiffViewer) View() string {
	if dv.diff == nil || len(dv.lines) == 0 {
		return "No diff to display. Select a file."
	}

	var b strings.Builder

	end := dv.offset + dv.height
	if end > len(dv.lines) {
		end = len(dv.lines)
	}

	for i := dv.offset; i < end; i++ {
		dl := dv.lines[i]
		isCursor := i == dv.cursor

		var line string
		if dl.isHunkHeader {
			line = hunkHeaderStyle.Render(dl.hunkHeader)
		} else {
			line = dv.renderCodeLine(dl, i)
		}

		if isCursor {
			line = cursorStyle.Render("→ ") + line
		} else {
			line = "  " + line
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func (dv DiffViewer) renderCodeLine(dl diffLine, idx int) string {
	l := dl.line
	oldNo := "     "
	newNo := "     "
	if l.OldLineNo > 0 {
		oldNo = fmt.Sprintf("%4d ", l.OldLineNo)
	}
	if l.NewLineNo > 0 {
		newNo = fmt.Sprintf("%4d ", l.NewLineNo)
	}

	gutter := lineNoStyle.Render(oldNo) + lineNoStyle.Render(newNo)

	marker := " "
	if dv.commentLines[idx] {
		marker = commentMarker
	}

	var content string
	switch l.Type {
	case git.LineAdded:
		content = addedLineStyle.Render("+" + l.Content)
	case git.LineRemoved:
		content = removedLineStyle.Render("-" + l.Content)
	default:
		content = contextLineStyle.Render(" " + l.Content)
	}

	return gutter + marker + " " + content
}

// CursorLine returns the current cursor position.
func (dv DiffViewer) CursorLine() int {
	return dv.cursor
}

// CurrentLine returns the git.Line at the cursor, or nil if on a hunk header.
func (dv DiffViewer) CurrentLine() *git.Line {
	if dv.cursor >= 0 && dv.cursor < len(dv.lines) {
		return dv.lines[dv.cursor].line
	}
	return nil
}

// CurrentLineNo returns the relevant line number for commenting (new line for added/context, old for removed).
func (dv DiffViewer) CurrentLineNo() int {
	l := dv.CurrentLine()
	if l == nil {
		return 0
	}
	if l.Type == git.LineRemoved {
		return l.OldLineNo
	}
	return l.NewLineNo
}

// SetFocused sets whether this component has focus.
func (dv *DiffViewer) SetFocused(focused bool) {
	dv.focused = focused
}

// SetSize updates the dimensions.
func (dv *DiffViewer) SetSize(width, height int) {
	dv.width = width
	dv.height = height
}

// TotalLines returns the total number of flattened lines.
func (dv DiffViewer) TotalLines() int {
	return len(dv.lines)
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/ui/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/diffview.go internal/ui/diffview_test.go
git commit -m "feat: add diff viewer sub-model with unified view and vim navigation"
```

---

## Task 9: Comment Input Sub-Model

**Files:**
- Create: `internal/ui/commentinput.go`
- Create: `internal/ui/commentinput_test.go`

**Step 1: Write tests**

Create `internal/ui/commentinput_test.go`:

```go
package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommentInputActivate(t *testing.T) {
	ci := NewCommentInput(80)

	if ci.Active() {
		t.Error("should not be active initially")
	}

	ci.Activate("main.go", 10, "")
	if !ci.Active() {
		t.Error("should be active after Activate")
	}
	if ci.FilePath() != "main.go" {
		t.Errorf("file = %q, want %q", ci.FilePath(), "main.go")
	}
	if ci.LineNo() != 10 {
		t.Errorf("line = %d, want 10", ci.LineNo())
	}
}

func TestCommentInputCancel(t *testing.T) {
	ci := NewCommentInput(80)
	ci.Activate("main.go", 10, "")

	ci, _ = ci.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if ci.Active() {
		t.Error("should be inactive after Esc")
	}
}

func TestCommentInputEditExisting(t *testing.T) {
	ci := NewCommentInput(80)
	ci.Activate("main.go", 10, "existing comment")

	if ci.Value() != "existing comment" {
		t.Errorf("value = %q, want %q", ci.Value(), "existing comment")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/ -v -run TestCommentInput
```

Expected: FAIL — NewCommentInput not defined

**Step 3: Implement comment input**

Create `internal/ui/commentinput.go`:

```go
package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var commentInputStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("3")).
	Padding(0, 1)

// CommentSubmitMsg is sent when the user submits a comment.
type CommentSubmitMsg struct {
	FilePath string
	LineNo   int
	Body     string
}

// CommentCancelMsg is sent when the user cancels comment input.
type CommentCancelMsg struct{}

// CommentInput is a sub-model for entering review comments.
type CommentInput struct {
	input    textinput.Model
	active   bool
	filePath string
	lineNo   int
	width    int
}

// NewCommentInput creates a new comment input component.
func NewCommentInput(width int) CommentInput {
	ti := textinput.New()
	ti.Placeholder = "Enter comment..."
	ti.CharLimit = 500
	ti.Width = width - 6

	return CommentInput{
		input: ti,
		width: width,
	}
}

// Activate shows the input and optionally pre-fills with an existing comment.
func (ci *CommentInput) Activate(filePath string, lineNo int, existing string) {
	ci.active = true
	ci.filePath = filePath
	ci.lineNo = lineNo
	ci.input.SetValue(existing)
	ci.input.Focus()
}

// Init returns the text input blink command.
func (ci CommentInput) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles key messages.
func (ci CommentInput) Update(msg tea.Msg) (CommentInput, tea.Cmd) {
	if !ci.active {
		return ci, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			ci.active = false
			ci.input.Blur()
			return ci, func() tea.Msg { return CommentCancelMsg{} }
		case tea.KeyEnter:
			body := ci.input.Value()
			ci.active = false
			ci.input.Blur()
			if body == "" {
				return ci, func() tea.Msg { return CommentCancelMsg{} }
			}
			fp := ci.filePath
			ln := ci.lineNo
			return ci, func() tea.Msg {
				return CommentSubmitMsg{
					FilePath: fp,
					LineNo:   ln,
					Body:     body,
				}
			}
		}
	}

	var cmd tea.Cmd
	ci.input, cmd = ci.input.Update(msg)
	return ci, cmd
}

// View renders the comment input.
func (ci CommentInput) View() string {
	if !ci.active {
		return ""
	}
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Comment: ")
	return commentInputStyle.Render(label + ci.input.View())
}

// Active returns whether the input is currently shown.
func (ci CommentInput) Active() bool {
	return ci.active
}

// FilePath returns the file being commented on.
func (ci CommentInput) FilePath() string {
	return ci.filePath
}

// LineNo returns the line being commented on.
func (ci CommentInput) LineNo() int {
	return ci.lineNo
}

// Value returns the current input text.
func (ci CommentInput) Value() string {
	return ci.input.Value()
}

// SetWidth updates the width.
func (ci *CommentInput) SetWidth(width int) {
	ci.width = width
	ci.input.Width = width - 6
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/ui/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/commentinput.go internal/ui/commentinput_test.go
git commit -m "feat: add comment input sub-model with submit and cancel"
```

---

## Task 10: Root Model — Orchestration

**Files:**
- Create: `internal/ui/root.go`
- Create: `internal/ui/root_test.go`
- Modify: `cmd/revui/main.go`

This is the largest task. The root model wires together file list, diff viewer, comment input, and the status bar.

**Step 1: Write tests**

Create `internal/ui/root_test.go`:

```go
package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/deparker/revui/internal/git"
)

type mockGitRunner struct {
	files []git.ChangedFile
	diffs map[string]*git.FileDiff
}

func (m *mockGitRunner) ChangedFiles(_ string) ([]git.ChangedFile, error) {
	return m.files, nil
}

func (m *mockGitRunner) FileDiff(_ string, path string) (*git.FileDiff, error) {
	if d, ok := m.diffs[path]; ok {
		return d, nil
	}
	return &git.FileDiff{Path: path}, nil
}

func (m *mockGitRunner) CurrentBranch() (string, error) {
	return "feature", nil
}

func newTestRoot() RootModel {
	mock := &mockGitRunner{
		files: []git.ChangedFile{
			{Path: "main.go", Status: "M"},
			{Path: "util.go", Status: "A"},
		},
		diffs: map[string]*git.FileDiff{
			"main.go": makeTestDiff(),
		},
	}
	return NewRootModel(mock, "main", 80, 24)
}

func TestRootFocusSwitching(t *testing.T) {
	m := newTestRoot()

	if m.focus != focusFileList {
		t.Errorf("initial focus = %d, want focusFileList", m.focus)
	}

	// l switches to diff viewer
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.focus != focusDiffViewer {
		t.Errorf("after l: focus = %d, want focusDiffViewer", m.focus)
	}

	// h switches back to file list
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.focus != focusFileList {
		t.Errorf("after h: focus = %d, want focusFileList", m.focus)
	}
}

func TestRootQuit(t *testing.T) {
	m := newTestRoot()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestRootViewNotEmpty(t *testing.T) {
	m := newTestRoot()
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/ -v -run TestRoot
```

Expected: FAIL — NewRootModel not defined

**Step 3: Implement root model**

Create `internal/ui/root.go`:

```go
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/comment"
	"github.com/deparker/revui/internal/git"
)

type focusArea int

const (
	focusFileList focusArea = iota
	focusDiffViewer
	focusCommentInput
)

// GitRunner is the interface for git operations, enabling testing with mocks.
type GitRunner interface {
	ChangedFiles(base string) ([]git.ChangedFile, error)
	FileDiff(base, path string) (*git.FileDiff, error)
	CurrentBranch() (string, error)
}

// finishMsg signals the review is done and comments should be copied.
type finishMsg struct{}

// RootModel is the top-level Bubble Tea model.
type RootModel struct {
	git          GitRunner
	base         string
	branch       string
	files        []git.ChangedFile
	fileList     FileList
	diffViewer   DiffViewer
	commentInput CommentInput
	comments     *comment.Store
	focus        focusArea
	width        int
	height       int
	err          error
	quitting     bool
	finished     bool
	output       string // formatted comments for clipboard
	fileListWidth int
}

// NewRootModel creates the root model with the given git runner and base branch.
func NewRootModel(gitRunner GitRunner, base string, width, height int) RootModel {
	fileListWidth := 30

	files, err := gitRunner.ChangedFiles(base)
	if err != nil {
		return RootModel{err: err}
	}

	branch, _ := gitRunner.CurrentBranch()

	fl := NewFileList(files, fileListWidth, height-2)
	dv := NewDiffViewer(width-fileListWidth-3, height-2)
	ci := NewCommentInput(width)

	// Load the first file's diff if available
	if len(files) > 0 {
		if fd, err := gitRunner.FileDiff(base, files[0].Path); err == nil {
			dv.SetDiff(fd)
		}
	}

	return RootModel{
		git:           gitRunner,
		base:          base,
		branch:        branch,
		files:         files,
		fileList:      fl,
		diffViewer:    dv,
		commentInput:  ci,
		comments:      comment.NewStore(),
		focus:         focusFileList,
		width:         width,
		height:        height,
		fileListWidth: fileListWidth,
	}
}

// Init returns the initial command.
func (m RootModel) Init() tea.Cmd {
	return nil
}

// Update handles all messages.
func (m RootModel) Update(msg tea.Msg) (RootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.fileList.SetSize(m.fileListWidth, m.height-2)
		m.diffViewer.SetSize(m.width-m.fileListWidth-3, m.height-2)
		m.commentInput.SetWidth(m.width)
		return m, nil

	case CommentSubmitMsg:
		line := m.diffViewer.CurrentLine()
		lineType := git.LineContext
		snippet := ""
		if line != nil {
			lineType = line.Type
			snippet = line.Content
		}
		m.comments.Add(comment.Comment{
			FilePath:    msg.FilePath,
			StartLine:   msg.LineNo,
			EndLine:     msg.LineNo,
			LineType:    lineType,
			Body:        msg.Body,
			CodeSnippet: snippet,
		})
		m.focus = focusDiffViewer
		m.updateCommentMarkers()
		return m, nil

	case CommentCancelMsg:
		m.focus = focusDiffViewer
		return m, nil

	case finishMsg:
		m.output = comment.Format(m.comments.All())
		m.finished = true
		return m, tea.Quit

	case tea.KeyMsg:
		// Comment input gets priority when active
		if m.focus == focusCommentInput {
			var cmd tea.Cmd
			m.commentInput, cmd = m.commentInput.Update(msg)
			return m, cmd
		}

		return m.handleKeyMsg(msg)
	}

	return m, nil
}

func (m RootModel) handleKeyMsg(msg tea.KeyMsg) (RootModel, tea.Cmd) {
	key := msg.String()

	switch key {
	case "q":
		m.quitting = true
		return m, tea.Quit

	case "Z":
		// ZZ to finish — we track the first Z and wait for the second
		// For simplicity, just use a single key combo check
		return m, nil

	case "ctrl+d":
		if m.focus == focusDiffViewer {
			m.diffViewer, _ = m.diffViewer.Update(msg)
		}
		return m, nil

	case "ctrl+u":
		if m.focus == focusDiffViewer {
			m.diffViewer, _ = m.diffViewer.Update(msg)
		}
		return m, nil

	case "h":
		if m.focus == focusDiffViewer {
			m.focus = focusFileList
		}
		return m, nil

	case "l", "enter":
		if m.focus == focusFileList {
			m.focus = focusDiffViewer
			// Load diff for selected file
			sel := m.fileList.SelectedFile()
			if fd, err := m.git.FileDiff(m.base, sel.Path); err == nil {
				m.diffViewer.SetDiff(fd)
				m.updateCommentMarkers()
			}
		}
		return m, nil

	case "c":
		if m.focus == focusDiffViewer {
			line := m.diffViewer.CurrentLine()
			if line != nil {
				lineNo := m.diffViewer.CurrentLineNo()
				sel := m.fileList.SelectedFile()
				existing := ""
				if c := m.comments.Get(sel.Path, lineNo); c != nil {
					existing = c.Body
				}
				m.commentInput.Activate(sel.Path, lineNo, existing)
				m.focus = focusCommentInput
			}
		}
		return m, nil

	case "D":
		// dd to delete — simplified to single D for now
		if m.focus == focusDiffViewer {
			lineNo := m.diffViewer.CurrentLineNo()
			sel := m.fileList.SelectedFile()
			m.comments.Delete(sel.Path, lineNo)
			m.updateCommentMarkers()
		}
		return m, nil
	}

	// Route to focused sub-model
	switch m.focus {
	case focusFileList:
		var cmd tea.Cmd
		m.fileList, cmd = m.fileList.Update(msg)
		// Auto-load diff when selection changes
		if key == "j" || key == "k" || key == "G" || key == "g" {
			sel := m.fileList.SelectedFile()
			if fd, err := m.git.FileDiff(m.base, sel.Path); err == nil {
				m.diffViewer.SetDiff(fd)
				m.updateCommentMarkers()
			}
		}
		return m, cmd

	case focusDiffViewer:
		var cmd tea.Cmd
		m.diffViewer, cmd = m.diffViewer.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *RootModel) updateCommentMarkers() {
	sel := m.fileList.SelectedFile()
	markers := make(map[int]bool)
	for _, c := range m.comments.ForFile(sel.Path) {
		// Find the flattened line index for this comment's line number
		for i := 0; i < m.diffViewer.TotalLines(); i++ {
			if lineNo := m.diffViewer.CurrentLineNo(); lineNo == c.StartLine {
				markers[i] = true
			}
		}
	}
	m.diffViewer.SetCommentLines(markers)
}

// View renders the full UI.
func (m RootModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Render(fmt.Sprintf(" revui — %s → %s ", m.base, m.branch))

	// File list panel
	fileListPanel := lipgloss.NewStyle().
		Width(m.fileListWidth).
		Height(m.height - 3).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Render(m.fileList.View())

	// Diff viewer panel
	diffPanel := lipgloss.NewStyle().
		Width(m.width - m.fileListWidth - 3).
		Height(m.height - 3).
		Render(m.diffViewer.View())

	// Main content
	content := lipgloss.JoinHorizontal(lipgloss.Top, fileListPanel, diffPanel)

	// Status bar
	statusBar := m.renderStatusBar()

	// Comment input (overlays status bar when active)
	if m.commentInput.Active() {
		statusBar = m.commentInput.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}

func (m RootModel) renderStatusBar() string {
	commentCount := len(m.comments.All())
	status := fmt.Sprintf(" [c]omment  [v]isual  [Tab]view  [q]uit  [ZZ]done  [?]help  │  %d comments", commentCount)

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(status)
}

// Output returns the formatted comment output (available after finish).
func (m RootModel) Output() string {
	return m.output
}

// Finished returns whether the review was completed (not quit).
func (m RootModel) Finished() bool {
	return m.finished
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/ui/ -v
```

Expected: PASS

**Step 5: Update main.go to wire everything together**

Update `cmd/revui/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/deparker/revui/internal/git"
	"github.com/deparker/revui/internal/ui"
)

func main() {
	base := flag.String("base", "main", "base branch to diff against")
	flag.Parse()

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	runner := &git.Runner{Dir: dir}
	if !runner.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "Error: not a git repository")
		os.Exit(1)
	}

	if !runner.BranchExists(*base) {
		fmt.Fprintf(os.Stderr, "Error: base branch %q does not exist. Use --base to specify.\n", *base)
		os.Exit(1)
	}

	model := ui.NewRootModel(runner, *base, 80, 24)

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	rm := finalModel.(ui.RootModel)
	if rm.Finished() && rm.Output() != "" {
		// TODO: copy to clipboard in a future task
		fmt.Print(rm.Output())
	}
}
```

Note: The `RootModel` needs to implement `tea.Model` interface. Update `root.go` Update method signature:

The `Update` method returns `(RootModel, tea.Cmd)` but `tea.Model` requires `(tea.Model, tea.Cmd)`. We need a wrapper. Add to `root.go`:

```go
// Ensure RootModel implements tea.Model by wrapping Update.
// The actual Update returns RootModel for testability.
// tea.Program will use the tea.Model interface.

// UpdateModel wraps Update for the tea.Model interface.
func (m RootModel) UpdateModel(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated, cmd
}
```

Actually, simpler: just change the Update signature to return `(tea.Model, tea.Cmd)` and adjust tests to type-assert. Update `root.go` `Update` method to return `(tea.Model, tea.Cmd)`, and update `root_test.go` to type-assert:

In tests, change:
```go
m, _ = m.Update(msg)
```
to:
```go
updated, _ := m.Update(msg)
m = updated.(RootModel)
```

**Step 6: Run all tests**

```bash
go test ./... -v
```

Expected: PASS

**Step 7: Verify it compiles**

```bash
go build ./cmd/revui
```

Expected: compiles successfully

**Step 8: Commit**

```bash
git add internal/ui/root.go internal/ui/root_test.go cmd/revui/main.go
git commit -m "feat: add root model orchestration and wire up main entry point"
```

---

## Task 11: Clipboard Integration

**Files:**
- Modify: `cmd/revui/main.go`

**Step 1: Install clipboard dependency**

```bash
go get github.com/atotto/clipboard
```

**Step 2: Update main.go to copy to clipboard**

Add clipboard copy after the program finishes:

```go
import "github.com/atotto/clipboard"

// In main(), after p.Run():
if rm.Finished() && rm.Output() != "" {
    if err := clipboard.WriteAll(rm.Output()); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: could not copy to clipboard: %v\n", err)
        fmt.Fprintf(os.Stderr, "Printing to stdout instead:\n\n")
        fmt.Print(rm.Output())
    } else {
        fmt.Println("Review comments copied to clipboard.")
    }
}
```

**Step 3: Verify it compiles**

```bash
go build ./cmd/revui
```

Expected: compiles

**Step 4: Commit**

```bash
git add cmd/revui/main.go go.mod go.sum
git commit -m "feat: copy review comments to clipboard on finish"
```

---

## Task 12: ZZ Key Sequence for Finish

**Files:**
- Modify: `internal/ui/root.go`
- Modify: `internal/ui/root_test.go`

The `ZZ` key sequence requires tracking that the first `Z` was pressed.

**Step 1: Write test**

Add to `root_test.go`:

```go
func TestRootZZFinish(t *testing.T) {
	m := newTestRoot()

	// First Z — no quit
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m = updated.(RootModel)
	if cmd != nil {
		t.Error("first Z should not produce a command")
	}

	// Second Z — should trigger finish
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m = updated.(RootModel)
	if !m.Finished() {
		t.Error("ZZ should trigger finish")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/ui/ -v -run TestRootZZ
```

Expected: FAIL

**Step 3: Add `pendingZ` state to RootModel and handle the sequence**

Add `pendingZ bool` field to `RootModel`. In `handleKeyMsg`, when `Z` is pressed and `pendingZ` is false, set it to true. When `Z` is pressed and `pendingZ` is true, trigger finish. Any other key resets `pendingZ`.

**Step 4: Run tests**

```bash
go test ./internal/ui/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/root.go internal/ui/root_test.go
git commit -m "feat: add ZZ key sequence to finish review"
```

---

## Task 13: Integrate Syntax Highlighting into Diff Viewer

**Files:**
- Modify: `internal/ui/diffview.go`
- Modify: `internal/ui/diffview_test.go`

**Step 1: Write test**

Add to `diffview_test.go`:

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

**Step 2: Add highlighting flag and integrate with the Highlighter**

Add a `highlighter *syntax.Highlighter` field and `highlightEnabled bool` to `DiffViewer`. In `renderCodeLine`, if highlighting is enabled, run the content through the highlighter.

**Step 3: Run tests**

```bash
go test ./internal/ui/ -v
```

Expected: PASS

**Step 4: Commit**

```bash
git add internal/ui/diffview.go internal/ui/diffview_test.go
git commit -m "feat: integrate chroma syntax highlighting into diff viewer"
```

---

## Task 14: Side-by-Side Diff View

**Files:**
- Create: `internal/ui/sidebyside.go`
- Create: `internal/ui/sidebyside_test.go`
- Modify: `internal/ui/diffview.go`

**Step 1: Write tests**

Create `internal/ui/sidebyside_test.go`:

```go
package ui

import (
	"testing"

	"github.com/deparker/revui/internal/git"
)

func TestSideBySideRender(t *testing.T) {
	lines := []git.Line{
		{Content: "old line", Type: git.LineRemoved, OldLineNo: 1},
		{Content: "new line", Type: git.LineAdded, NewLineNo: 1},
		{Content: "context", Type: git.LineContext, OldLineNo: 2, NewLineNo: 2},
	}

	pairs := BuildSideBySidePairs(lines)
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}

	// First pair: removed + added
	if pairs[0].Left == nil || pairs[0].Right == nil {
		t.Error("first pair should have both sides")
	}

	// Second pair: context on both sides
	if pairs[1].Left == nil || pairs[1].Right == nil {
		t.Error("second pair should have both sides for context")
	}
}
```

**Step 2: Implement side-by-side pairing logic**

Create `internal/ui/sidebyside.go` with a `LinePair` struct and `BuildSideBySidePairs` function that pairs removed/added lines and maps context lines to both sides.

**Step 3: Add `Tab` toggle to DiffViewer to switch between unified and side-by-side**

Add a `sideBySide bool` field. When `Tab` is pressed, toggle it. In `View`, delegate to a side-by-side renderer when enabled.

**Step 4: Run tests**

```bash
go test ./internal/ui/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/sidebyside.go internal/ui/sidebyside_test.go internal/ui/diffview.go
git commit -m "feat: add toggleable side-by-side diff view"
```

---

## Task 15: Visual Mode for Range Selection

**Files:**
- Modify: `internal/ui/diffview.go`
- Modify: `internal/ui/diffview_test.go`

**Step 1: Write tests**

Add to `diffview_test.go`:

```go
func TestDiffViewVisualMode(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())

	// Enter visual mode with v
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !dv.InVisualMode() {
		t.Error("should be in visual mode after v")
	}

	// Move down to extend selection
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	start, end := dv.VisualRange()
	if start != 0 || end != 2 {
		t.Errorf("range = %d-%d, want 0-2", start, end)
	}

	// Esc exits visual mode
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if dv.InVisualMode() {
		t.Error("should not be in visual mode after Esc")
	}
}
```

**Step 2: Add visual mode state**

Add `visualMode bool`, `visualStart int` fields to `DiffViewer`. When `v` is pressed, enable visual mode and record start position. Navigation while in visual mode extends the selection. `Esc` cancels. Render selected lines with a distinct style.

**Step 3: Run tests**

```bash
go test ./internal/ui/ -v
```

Expected: PASS

**Step 4: Commit**

```bash
git add internal/ui/diffview.go internal/ui/diffview_test.go
git commit -m "feat: add visual mode for line range selection"
```

---

## Task 16: Comment Navigation (`]c` / `[c`)

**Files:**
- Modify: `internal/ui/diffview.go`
- Modify: `internal/ui/diffview_test.go`

**Step 1: Write tests**

```go
func TestDiffViewCommentNavigation(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())
	dv.SetCommentLines(map[int]bool{1: true, 4: true})

	// ]c jumps to next comment
	// Starting at 0, should jump to 1
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	// Need to handle two-key sequence ]c — simplify to ']' for now
	if dv.CursorLine() != 1 {
		t.Errorf("after ]c: cursor = %d, want 1", dv.CursorLine())
	}
}
```

**Step 2: Implement `]c` / `[c` navigation**

Track pending `]` or `[` key state, then on `c` jump to next/previous comment.

**Step 3: Run tests and commit**

```bash
go test ./internal/ui/ -v
git add internal/ui/diffview.go internal/ui/diffview_test.go
git commit -m "feat: add ]c/[c comment navigation in diff viewer"
```

---

## Task 17: Help Overlay

**Files:**
- Create: `internal/ui/help.go`
- Modify: `internal/ui/root.go`

**Step 1: Create help view**

Create `internal/ui/help.go` with a static help text rendering function that lists all keybindings.

**Step 2: Add `?` keybinding to root model**

Add a `showHelp bool` field. When `?` is pressed, toggle it. When active, overlay the help text on top of the main view. `Esc` or `?` dismisses it.

**Step 3: Run tests and commit**

```bash
go test ./... -v
git add internal/ui/help.go internal/ui/root.go
git commit -m "feat: add help overlay with keybinding reference"
```

---

## Task 18: Search in Diff (`/`, `n`, `N`)

**Files:**
- Modify: `internal/ui/diffview.go`
- Create: `internal/ui/search.go`
- Modify: `internal/ui/diffview_test.go`

**Step 1: Write tests for search**

```go
func TestDiffViewSearch(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())

	// Search for "new line"
	dv.SetSearch("new line")
	matches := dv.SearchMatches()
	if len(matches) == 0 {
		t.Error("expected search matches")
	}
}
```

**Step 2: Implement search**

Add search state to `DiffViewer`: `searchTerm string`, `searchMatches []int`, `searchIdx int`. When `/` is pressed in root model, show a search input. `n`/`N` jump between matches. Highlight matching lines.

**Step 3: Run tests and commit**

```bash
go test ./... -v
git add internal/ui/
git commit -m "feat: add / search with n/N navigation in diff viewer"
```

---

## Task 19: End-to-End Manual Testing

**Files:** None new — verification only

**Step 1: Build the binary**

```bash
go build -o revui ./cmd/revui
```

**Step 2: Test in a real git repo**

```bash
cd /some/repo/with/a/feature/branch
/home/deparker/Code/revui/revui --base main
```

Verify:
- [ ] File list shows changed files
- [ ] Selecting a file shows its diff
- [ ] j/k navigation works in both panels
- [ ] h/l switches focus between panels
- [ ] Syntax highlighting is visible
- [ ] `c` opens comment input
- [ ] Comment shows marker in gutter
- [ ] `Tab` toggles side-by-side view
- [ ] `ZZ` copies comments to clipboard
- [ ] `q` quits without copying
- [ ] `?` shows help

**Step 3: Fix any issues found and commit**

```bash
go test ./... -v
git add .
git commit -m "fix: address issues found during manual testing"
```

---

## Summary

| Task | Description | Dependencies |
|------|-------------|-------------|
| 1 | Project scaffolding | — |
| 2 | Git data models | 1 |
| 3 | Diff parser | 2 |
| 4 | Git command runner | 3 |
| 5 | Comment model + formatter | 2 |
| 6 | Syntax highlighting | 1 |
| 7 | File list sub-model | 2 |
| 8 | Diff viewer sub-model | 2 |
| 9 | Comment input sub-model | 1 |
| 10 | Root model (orchestration) | 4, 5, 7, 8, 9 |
| 11 | Clipboard integration | 10 |
| 12 | ZZ key sequence | 10 |
| 13 | Syntax highlighting integration | 6, 8 |
| 14 | Side-by-side diff view | 8 |
| 15 | Visual mode | 8 |
| 16 | Comment navigation | 8, 10 |
| 17 | Help overlay | 10 |
| 18 | Search in diff | 8 |
| 19 | End-to-end testing | all |
