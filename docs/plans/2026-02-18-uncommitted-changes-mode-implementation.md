# Uncommitted Changes Mode Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Auto-detect uncommitted changes and show them for review instead of branch comparison; fall back to branch mode when clean.

**Architecture:** New git methods (`HasUncommittedChanges`, `UncommittedFiles`, `UncommittedFileDiff`) handle detection, file listing, and diff generation including untracked and binary files. A `reviewMode` enum on `RootModel` switches between branch and uncommitted behavior. Detection happens in `main.go` before model construction.

**Tech Stack:** Go, Bubble Tea, `os/exec` for git commands, `os` for file reading.

---

### Task 1: Add `HasUncommittedChanges` to git package

**Files:**
- Modify: `internal/git/git.go`
- Test: `internal/git/git_test.go`

**Step 1: Write the failing test**

Add to `internal/git/git_test.go`:

```go
func TestHasUncommittedChanges(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}

	// Repo is clean after setupTestRepo (all committed)
	if r.HasUncommittedChanges() {
		t.Error("expected no uncommitted changes in clean repo")
	}

	// Create an unstaged change
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n\nfunc hello() { /* modified */ }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !r.HasUncommittedChanges() {
		t.Error("expected uncommitted changes after modifying a file")
	}

	// Stage the change - still uncommitted
	runCmd(t, dir, "git", "add", "hello.go")
	if !r.HasUncommittedChanges() {
		t.Error("expected uncommitted changes after staging")
	}

	// Commit - now clean again
	runCmd(t, dir, "git", "commit", "-m", "fix")
	if r.HasUncommittedChanges() {
		t.Error("expected no uncommitted changes after commit")
	}

	// Untracked file counts too
	if err := os.WriteFile(filepath.Join(dir, "newfile.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !r.HasUncommittedChanges() {
		t.Error("expected uncommitted changes with untracked file")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/git/ -run TestHasUncommittedChanges -v`
Expected: FAIL — `HasUncommittedChanges` not defined.

**Step 3: Write minimal implementation**

Add to `internal/git/git.go`:

```go
// HasUncommittedChanges returns true if there are staged, unstaged, or untracked changes.
func (r *Runner) HasUncommittedChanges() bool {
	out, err := r.run("status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/git/ -run TestHasUncommittedChanges -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "feat: add HasUncommittedChanges to git runner"
```

---

### Task 2: Add `UncommittedFiles` to git package

**Files:**
- Modify: `internal/git/git.go`
- Test: `internal/git/git_test.go`

**Step 1: Write the failing test**

Add to `internal/git/git_test.go`:

```go
func TestUncommittedFiles(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}

	// Modify a tracked file (unstaged)
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n\nfunc hello() { /* changed */ }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Add a new untracked file
	if err := os.WriteFile(filepath.Join(dir, "untracked.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := r.UncommittedFiles()
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, f := range files {
		got[f.Path] = f.Status
	}

	if got["hello.go"] != "M" {
		t.Errorf("hello.go status = %q, want M", got["hello.go"])
	}
	if got["untracked.go"] != "A" {
		t.Errorf("untracked.go status = %q, want A", got["untracked.go"])
	}
}

func TestUncommittedFilesBinary(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}

	// Create a binary file (contains null bytes)
	if err := os.WriteFile(filepath.Join(dir, "image.png"), []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00}, 0644); err != nil {
		t.Fatal(err)
	}

	files, err := r.UncommittedFiles()
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, f := range files {
		got[f.Path] = f.Status
	}

	if got["image.png"] != "B" {
		t.Errorf("image.png status = %q, want B", got["image.png"])
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/git/ -run TestUncommittedFiles -v`
Expected: FAIL — `UncommittedFiles` not defined.

**Step 3: Write minimal implementation**

Add to `internal/git/git.go`:

```go
// UncommittedFiles returns changed files (staged + unstaged vs HEAD) plus untracked files.
// Binary files are marked with status "B".
func (r *Runner) UncommittedFiles() ([]ChangedFile, error) {
	// Get tracked changes (staged + unstaged)
	diffOut, err := r.run("diff", "HEAD", "--name-status")
	if err != nil {
		// If HEAD doesn't exist (initial commit), try --cached
		diffOut, err = r.run("diff", "--cached", "--name-status")
		if err != nil {
			diffOut = ""
		}
	}
	files := ParseNameStatus(diffOut)

	// Identify binary files among tracked changes via --numstat
	binaries := r.detectBinaryTracked()

	// Mark binary tracked files
	for i := range files {
		if binaries[files[i].Path] {
			files[i].Status = "B"
		}
	}

	// Get untracked files
	untrackedOut, err := r.run("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return files, nil
	}

	seen := make(map[string]bool, len(files))
	for _, f := range files {
		seen[f.Path] = true
	}

	for line := range strings.SplitSeq(strings.TrimSpace(untrackedOut), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		status := "A"
		if r.isBinaryFile(line) {
			status = "B"
		}
		files = append(files, ChangedFile{Path: line, Status: status})
	}

	return files, nil
}

// detectBinaryTracked returns a set of paths that are binary among tracked changes.
func (r *Runner) detectBinaryTracked() map[string]bool {
	out, err := r.run("diff", "HEAD", "--numstat")
	if err != nil {
		return nil
	}
	binaries := make(map[string]bool)
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Binary files show as "-\t-\tfilename"
		if strings.HasPrefix(line, "-\t-\t") {
			path := line[4:]
			binaries[path] = true
		}
	}
	return binaries
}

// isBinaryFile checks if a file appears to be binary by looking for null bytes in the first 8KB.
func (r *Runner) isBinaryFile(path string) bool {
	fullPath := filepath.Join(r.Dir, path)
	f, err := os.Open(fullPath)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 8192)
	n, err := f.Read(buf)
	if n == 0 {
		return false
	}
	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}
```

Note: This requires adding `"os"` and `"path/filepath"` to imports in `git.go`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/git/ -run TestUncommittedFiles -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "feat: add UncommittedFiles to git runner"
```

---

### Task 3: Add `UncommittedFileDiff` to git package

**Files:**
- Modify: `internal/git/git.go`
- Test: `internal/git/git_test.go`

**Step 1: Write the failing test**

Add to `internal/git/git_test.go`:

```go
func TestUncommittedFileDiff(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}

	// Modify a tracked file
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n\nfunc hello() { /* changed */ }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	fd, err := r.UncommittedFileDiff("hello.go")
	if err != nil {
		t.Fatal(err)
	}
	if fd.Path != "hello.go" {
		t.Errorf("path = %q, want %q", fd.Path, "hello.go")
	}
	if len(fd.Hunks) == 0 {
		t.Fatal("expected at least one hunk for modified tracked file")
	}
}

func TestUncommittedFileDiffUntracked(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}

	content := "package main\n\nfunc newFunc() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "brand_new.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fd, err := r.UncommittedFileDiff("brand_new.go")
	if err != nil {
		t.Fatal(err)
	}
	if fd.Path != "brand_new.go" {
		t.Errorf("path = %q, want %q", fd.Path, "brand_new.go")
	}
	if len(fd.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(fd.Hunks))
	}
	// All lines should be added
	for _, line := range fd.Hunks[0].Lines {
		if line.Type != LineAdded {
			t.Errorf("expected all lines to be LineAdded, got %v for %q", line.Type, line.Content)
		}
	}
	// Line numbers should start at 1
	if fd.Hunks[0].Lines[0].NewLineNo != 1 {
		t.Errorf("first line number = %d, want 1", fd.Hunks[0].Lines[0].NewLineNo)
	}
}

func TestUncommittedFileDiffBinary(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}

	if err := os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0xFF, 0x00, 0xAB}, 0644); err != nil {
		t.Fatal(err)
	}

	fd, err := r.UncommittedFileDiff("data.bin")
	if err != nil {
		t.Fatal(err)
	}
	if fd.Path != "data.bin" {
		t.Errorf("path = %q, want %q", fd.Path, "data.bin")
	}
	if fd.Status != "B" {
		t.Errorf("status = %q, want B", fd.Status)
	}
	if len(fd.Hunks) != 0 {
		t.Errorf("expected 0 hunks for binary file, got %d", len(fd.Hunks))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/git/ -run TestUncommittedFileDiff -v`
Expected: FAIL — `UncommittedFileDiff` not defined.

**Step 3: Write minimal implementation**

Add to `internal/git/git.go`:

```go
// UncommittedFileDiff returns the diff for a single file against HEAD.
// For untracked files, it synthesizes an all-added diff.
// For binary files, it returns a FileDiff with status "B" and no hunks.
func (r *Runner) UncommittedFileDiff(path string) (*FileDiff, error) {
	// Check if binary
	if r.isBinaryFile(path) {
		return &FileDiff{Path: path, Status: "B"}, nil
	}

	// Try diffing against HEAD (works for tracked files)
	out, err := r.run("diff", "HEAD", "--", path)
	if err != nil || strings.TrimSpace(out) == "" {
		// Likely untracked — synthesize an all-added diff
		return r.synthesizeNewFileDiff(path)
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

// synthesizeNewFileDiff reads a file and creates a FileDiff where every line is added.
func (r *Runner) synthesizeNewFileDiff(path string) (*FileDiff, error) {
	fullPath := filepath.Join(r.Dir, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading untracked file %s: %w", path, err)
	}

	rawLines := strings.Split(string(content), "\n")
	// Remove trailing empty line from final newline
	if len(rawLines) > 0 && rawLines[len(rawLines)-1] == "" {
		rawLines = rawLines[:len(rawLines)-1]
	}

	lines := make([]Line, len(rawLines))
	for i, l := range rawLines {
		lines[i] = Line{
			Content:   l,
			Type:      LineAdded,
			NewLineNo: i + 1,
		}
	}

	return &FileDiff{
		Path:   path,
		Status: "A",
		Hunks: []Hunk{
			{
				OldStart: 0,
				OldCount: 0,
				NewStart: 1,
				NewCount: len(lines),
				Header:   fmt.Sprintf("@@ -0,0 +1,%d @@", len(lines)),
				Lines:    lines,
			},
		},
	}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/git/ -run TestUncommittedFileDiff -v`
Expected: PASS

**Step 5: Run all git tests to ensure nothing broke**

Run: `go test ./internal/git/ -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "feat: add UncommittedFileDiff to git runner"
```

---

### Task 4: Update `GitRunner` interface and add review mode type

**Files:**
- Modify: `internal/ui/root.go`
- Modify: `internal/ui/root_test.go`

**Step 1: Update the `GitRunner` interface and add `reviewMode` type**

In `internal/ui/root.go`, add three methods to `GitRunner`:

```go
type GitRunner interface {
	ChangedFiles(base string) ([]git.ChangedFile, error)
	FileDiff(base, path string) (*git.FileDiff, error)
	CurrentBranch() (string, error)
	HasUncommittedChanges() bool
	UncommittedFiles() ([]git.ChangedFile, error)
	UncommittedFileDiff(path string) (*git.FileDiff, error)
}
```

Add the mode type before `RootModel`:

```go
type reviewMode int

const (
	modeBranch      reviewMode = iota
	modeUncommitted
)
```

Add a `mode` field to `RootModel`:

```go
type RootModel struct {
	// ... existing fields ...
	mode          reviewMode
}
```

**Step 2: Update `mockGitRunner` in tests**

In `internal/ui/root_test.go`, add stub methods to satisfy the interface:

```go
func (m *mockGitRunner) HasUncommittedChanges() bool {
	return false
}

func (m *mockGitRunner) UncommittedFiles() ([]git.ChangedFile, error) {
	return m.files, nil
}

func (m *mockGitRunner) UncommittedFileDiff(path string) (*git.FileDiff, error) {
	if d, ok := m.diffs[path]; ok {
		return d, nil
	}
	return &git.FileDiff{Path: path}, nil
}
```

**Step 3: Run tests to verify everything compiles and passes**

Run: `go test ./internal/ui/ -v`
Expected: All PASS (existing behavior unchanged)

**Step 4: Commit**

```bash
git add internal/ui/root.go internal/ui/root_test.go
git commit -m "feat: extend GitRunner interface with uncommitted methods"
```

---

### Task 5: Add `loadFileDiff` helper and uncommitted-mode constructor

**Files:**
- Modify: `internal/ui/root.go`
- Modify: `internal/ui/root_test.go`

**Step 1: Write the failing test for uncommitted mode**

Add to `internal/ui/root_test.go`:

```go
func newTestRootUncommitted() RootModel {
	mock := &mockGitRunner{
		files: []git.ChangedFile{
			{Path: "main.go", Status: "M"},
			{Path: "newfile.go", Status: "A"},
			{Path: "image.png", Status: "B"},
		},
		diffs: map[string]*git.FileDiff{
			"main.go": makeTestDiff(),
			"image.png": {Path: "image.png", Status: "B"},
		},
	}
	return NewRootModelUncommitted(mock, 80, 24)
}

func TestRootUncommittedHeader(t *testing.T) {
	m := newTestRootUncommitted()
	view := m.View()
	if !strings.Contains(view, "uncommitted changes") {
		t.Error("expected header to contain 'uncommitted changes'")
	}
	if strings.Contains(view, "→") {
		t.Error("uncommitted mode header should not contain '→'")
	}
}

func TestRootUncommittedFileList(t *testing.T) {
	m := newTestRootUncommitted()
	if len(m.files) != 3 {
		t.Errorf("expected 3 files, got %d", len(m.files))
	}
}
```

Note: Add `"strings"` to imports if not already present.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/ -run TestRootUncommitted -v`
Expected: FAIL — `NewRootModelUncommitted` not defined.

**Step 3: Write the implementation**

Add `NewRootModelUncommitted` to `internal/ui/root.go`:

```go
// NewRootModelUncommitted creates the root model for reviewing uncommitted changes.
func NewRootModelUncommitted(gitRunner GitRunner, width, height int) RootModel {
	fileListWidth := 30

	files, err := gitRunner.UncommittedFiles()
	if err != nil {
		return RootModel{err: err}
	}

	fl := NewFileList(files, fileListWidth, height-2)
	dv := NewDiffViewer(width-fileListWidth-3, height-2)
	ci := NewCommentInput(width)

	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 100
	si.Width = width - 10

	// Load the first file's diff if available
	if len(files) > 0 {
		if fd, err := gitRunner.UncommittedFileDiff(files[0].Path); err == nil {
			dv.SetDiff(fd)
		}
	}

	return RootModel{
		git:           gitRunner,
		mode:          modeUncommitted,
		files:         files,
		fileList:      fl,
		diffViewer:    dv,
		commentInput:  ci,
		searchInput:   si,
		comments:      comment.NewStore(),
		focus:         focusFileList,
		width:         width,
		height:        height,
		fileListWidth: fileListWidth,
	}
}
```

Add the `loadFileDiff` helper method:

```go
// loadFileDiff loads the diff for the given path based on the current review mode.
func (m *RootModel) loadFileDiff(path string) (*git.FileDiff, error) {
	if m.mode == modeUncommitted {
		return m.git.UncommittedFileDiff(path)
	}
	return m.git.FileDiff(m.base, path)
}
```

Now replace all 4 call sites in `root.go` that call `m.git.FileDiff(m.base, ...)` to use `m.loadFileDiff(...)` instead. The call sites are:

1. In `handleKeyMsg`, case `"l", "enter"`: change `m.git.FileDiff(m.base, sel.Path)` to `m.loadFileDiff(sel.Path)`
2. In `handleKeyMsg`, case routing `focusFileList` (j/k/G/g keys): change `m.git.FileDiff(m.base, sel.Path)` to `m.loadFileDiff(sel.Path)`
3. In `Update`, case `navigateFileMsg`: change `m.git.FileDiff(m.base, sel.Path)` to `m.loadFileDiff(sel.Path)`

Note: The initial load in `NewRootModel` already uses the right method for its mode, and `NewRootModelUncommitted` uses `UncommittedFileDiff`, so those don't need the helper.

Update the header in `View()`:

```go
// Header — replace the current header block:
var headerText string
if m.mode == modeUncommitted {
	headerText = " revui — uncommitted changes "
} else {
	headerText = fmt.Sprintf(" revui — %s → %s ", m.base, m.branch)
}
header := lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("12")).
	Render(headerText)
```

**Step 4: Run tests to verify**

Run: `go test ./internal/ui/ -v`
Expected: All PASS (new and existing tests)

**Step 5: Commit**

```bash
git add internal/ui/root.go internal/ui/root_test.go
git commit -m "feat: add uncommitted mode constructor and loadFileDiff helper"
```

---

### Task 6: Add binary file status icon and diff placeholder

**Files:**
- Modify: `internal/ui/filelist.go`
- Modify: `internal/ui/diffview.go`
- Modify: `internal/ui/diffview_test.go`

**Step 1: Write the failing test for binary diff placeholder**

Add to `internal/ui/diffview_test.go`:

```go
func TestDiffViewBinaryPlaceholder(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(&git.FileDiff{Path: "image.png", Status: "B"})

	view := dv.View()
	if !strings.Contains(view, "Binary file") {
		t.Error("expected binary file placeholder message")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/ -run TestDiffViewBinaryPlaceholder -v`
Expected: FAIL — currently shows "No diff to display" instead of "Binary file" message.

**Step 3: Implement the changes**

In `internal/ui/diffview.go`, update `View()` to check for binary before the empty-lines check:

```go
func (dv DiffViewer) View() string {
	if dv.diff != nil && dv.diff.Status == "B" {
		return "Binary file — cannot display diff"
	}
	if dv.diff == nil || len(dv.lines) == 0 {
		return "No diff to display. Select a file."
	}
	// ... rest unchanged
```

In `internal/ui/filelist.go`, add the binary case to `statusIcon`:

```go
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
	case "B":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render("B")
	default:
		return "?"
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/ui/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/ui/filelist.go internal/ui/diffview.go internal/ui/diffview_test.go
git commit -m "feat: add binary file status icon and diff placeholder"
```

---

### Task 7: Update `main.go` with auto-detection

**Files:**
- Modify: `cmd/revui/main.go`

**Step 1: Update `main.go`**

Replace the model construction logic to auto-detect uncommitted changes:

```go
func main() {
	base := flag.String("base", "", "base branch to diff against (auto-detected if not set)")
	remote := flag.String("remote", "origin", "remote to detect default branch from")
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

	var model ui.RootModel
	if runner.HasUncommittedChanges() {
		model = ui.NewRootModelUncommitted(runner, 80, 24)
	} else {
		baseBranch := *base
		if baseBranch == "" {
			baseBranch = runner.DefaultBranch(*remote)
		}
		if !runner.BranchExists(baseBranch) {
			fmt.Fprintf(os.Stderr, "Error: base branch %q does not exist. Use --base to specify.\n", baseBranch)
			os.Exit(1)
		}
		model = ui.NewRootModel(runner, baseBranch, 80, 24)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	// ... rest stays the same
```

**Step 2: Build and verify it compiles**

Run: `go build ./cmd/revui`
Expected: Compiles without errors.

**Step 3: Run all tests**

Run: `go test ./...`
Expected: All PASS

**Step 4: Commit**

```bash
git add cmd/revui/main.go
git commit -m "feat: auto-detect uncommitted changes in main"
```

---

### Task 8: Add `FileStatusString` case for binary and run full suite

**Files:**
- Modify: `internal/git/types.go`

**Step 1: Update `FileStatusString`**

Add the binary case to `internal/git/types.go`:

```go
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
	case "B":
		return "binary"
	default:
		return "unknown"
	}
}
```

**Step 2: Run all tests and vet**

Run: `go test ./... && go vet ./...`
Expected: All PASS, no vet warnings.

**Step 3: Commit**

```bash
git add internal/git/types.go
git commit -m "feat: add binary status to FileStatusString"
```

---

### Task 9: Manual smoke test

**Step 1: Build**

Run: `go build ./cmd/revui`

**Step 2: Test uncommitted mode**

Create a temporary change in the repo (don't commit), then run `./revui`. Verify:
- Header shows "uncommitted changes"
- Changed files appear in the file list
- Diffs render correctly
- Press `q` to quit

**Step 3: Test clean repo fallback**

Undo the temporary change, then run `./revui`. Verify:
- Normal branch comparison behavior
- Header shows "base → branch"

**Step 4: Test with a binary file**

Create a binary file (e.g., copy a small image), run `./revui`. Verify:
- Binary file appears in file list with `B` marker
- Selecting it shows "Binary file — cannot display diff"
- Pressing `c` on the binary file opens the comment input

**Step 5: Clean up temporary test files and commit any missed fixes**
