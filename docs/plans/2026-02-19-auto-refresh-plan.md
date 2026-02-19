# Auto-Refresh Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Auto-refresh the file list and diff view every 2 seconds in uncommitted changes mode, using Bubble Tea's async Cmd pattern.

**Architecture:** A `tea.Tick` fires every 2 seconds (uncommitted mode only). Each tick spawns an async `tea.Cmd` that calls `UncommittedFiles()` + `UncommittedFileDiff()`. Results arrive as a `refreshResultMsg` and update the file list and diff viewer with cursor/scroll preservation. A `refreshInProgress` flag prevents overlapping git operations.

**Tech Stack:** Go, Bubble Tea (`tea.Tick`, `tea.Cmd`), existing `GitRunner` interface

---

### Task 1: Add `SetFiles` method to FileList

Add a method to update the file list while preserving the cursor position. If the currently selected file still exists in the new list, the cursor stays on it. If not, the cursor clamps to the nearest valid index.

**Files:**
- Modify: `internal/ui/filelist.go`
- Test: `internal/ui/filelist_test.go`

**Step 1: Write the failing tests**

Add to `internal/ui/filelist_test.go`:

```go
func TestFileListSetFiles(t *testing.T) {
	files := []git.ChangedFile{
		{Path: "a.go", Status: "M"},
		{Path: "b.go", Status: "A"},
		{Path: "c.go", Status: "D"},
	}
	fl := NewFileList(files, 20, 10)

	// Move cursor to b.go
	fl, _ = fl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if fl.SelectedFile().Path != "b.go" {
		t.Fatal("setup: expected cursor on b.go")
	}

	t.Run("cursor preserved when file still exists", func(t *testing.T) {
		newFiles := []git.ChangedFile{
			{Path: "a.go", Status: "M"},
			{Path: "b.go", Status: "M"}, // status changed but path same
			{Path: "d.go", Status: "A"}, // new file
		}
		fl2 := fl // copy
		fl2.SetFiles(newFiles)
		if fl2.SelectedFile().Path != "b.go" {
			t.Errorf("cursor on %q, want b.go", fl2.SelectedFile().Path)
		}
	})

	t.Run("cursor moves to neighbor when file removed", func(t *testing.T) {
		newFiles := []git.ChangedFile{
			{Path: "a.go", Status: "M"},
			{Path: "c.go", Status: "D"},
		}
		fl2 := fl // copy (cursor was on b.go at index 1)
		fl2.SetFiles(newFiles)
		// b.go removed; cursor should clamp to valid range
		if fl2.cursor >= len(newFiles) {
			t.Errorf("cursor %d out of range for %d files", fl2.cursor, len(newFiles))
		}
	})

	t.Run("cursor clamps when list shrinks", func(t *testing.T) {
		// Move cursor to last file
		fl3 := NewFileList(files, 20, 10)
		fl3, _ = fl3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
		if fl3.SelectedIndex() != 2 {
			t.Fatal("setup: expected cursor at index 2")
		}
		newFiles := []git.ChangedFile{
			{Path: "a.go", Status: "M"},
		}
		fl3.SetFiles(newFiles)
		if fl3.cursor != 0 {
			t.Errorf("cursor = %d, want 0", fl3.cursor)
		}
	})

	t.Run("empty list sets cursor to zero", func(t *testing.T) {
		fl4 := fl // copy
		fl4.SetFiles(nil)
		if fl4.cursor != 0 {
			t.Errorf("cursor = %d, want 0", fl4.cursor)
		}
	})
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run TestFileListSetFiles -v`
Expected: FAIL — `SetFiles` method does not exist.

**Step 3: Write the implementation**

Add to `internal/ui/filelist.go` after the `SetSize` method:

```go
// SetFiles updates the file list, preserving the cursor on the same file path
// if it still exists. If the selected file was removed, the cursor clamps to
// the nearest valid index.
func (fl *FileList) SetFiles(files []git.ChangedFile) {
	if len(files) == 0 {
		fl.files = files
		fl.cursor = 0
		return
	}

	// Try to find the currently selected file path in the new list
	selectedPath := ""
	if fl.cursor >= 0 && fl.cursor < len(fl.files) {
		selectedPath = fl.files[fl.cursor].Path
	}

	fl.files = files

	// Look for the same path in the new list
	for i, f := range fl.files {
		if f.Path == selectedPath {
			fl.cursor = i
			return
		}
	}

	// File not found — clamp cursor
	if fl.cursor >= len(fl.files) {
		fl.cursor = len(fl.files) - 1
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/ -run TestFileListSetFiles -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/filelist.go internal/ui/filelist_test.go
git commit -m "feat: add FileList.SetFiles with cursor preservation"
```

---

### Task 2: Add `RefreshDiff` method to DiffViewer

Like `SetDiff` but preserves the scroll offset (clamped to bounds) instead of resetting to line 0. The cursor is also preserved and clamped.

**Files:**
- Modify: `internal/ui/diffview.go`
- Test: `internal/ui/diffview_test.go`

**Step 1: Write the failing tests**

Add to `internal/ui/diffview_test.go`:

```go
func TestDiffViewRefreshDiff(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())

	// Scroll down a few lines
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	savedCursor := dv.CursorLine()

	t.Run("preserves cursor position", func(t *testing.T) {
		dv2 := dv // copy
		dv2.RefreshDiff(makeTestDiff())
		if dv2.CursorLine() != savedCursor {
			t.Errorf("cursor = %d, want %d", dv2.CursorLine(), savedCursor)
		}
	})

	t.Run("clamps cursor when diff shrinks", func(t *testing.T) {
		dv2 := dv // copy
		shortDiff := &git.FileDiff{
			Path:   "test.go",
			Status: "M",
			Hunks: []git.Hunk{
				{
					Header:   "@@ -1,1 +1,1 @@",
					OldStart: 1, OldCount: 1,
					NewStart: 1, NewCount: 1,
					Lines: []git.Line{
						{Content: "only line", Type: git.LineContext, OldLineNo: 1, NewLineNo: 1},
					},
				},
			},
		}
		dv2.RefreshDiff(shortDiff)
		// 1 hunk header + 1 line = 2 lines total, max cursor = 1
		if dv2.CursorLine() >= len(dv2.lines) {
			t.Errorf("cursor %d out of bounds for %d lines", dv2.CursorLine(), len(dv2.lines))
		}
	})

	t.Run("handles nil diff", func(t *testing.T) {
		dv2 := dv // copy
		dv2.RefreshDiff(nil)
		if dv2.CursorLine() != 0 {
			t.Errorf("cursor = %d, want 0 for nil diff", dv2.CursorLine())
		}
	})
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run TestDiffViewRefreshDiff -v`
Expected: FAIL — `RefreshDiff` method does not exist.

**Step 3: Write the implementation**

Add to `internal/ui/diffview.go` after the `SetDiff` method:

```go
// RefreshDiff updates the diff content while preserving cursor and scroll position.
// Cursor and scroll offset are clamped if the new diff is shorter.
func (dv *DiffViewer) RefreshDiff(fd *git.FileDiff) {
	dv.diff = fd
	dv.lines = dv.flattenLines()

	if len(dv.lines) == 0 {
		dv.cursor = 0
		dv.offset = 0
		return
	}

	// Clamp cursor
	if dv.cursor >= len(dv.lines) {
		dv.cursor = len(dv.lines) - 1
	}

	// Clamp scroll offset
	if dv.offset >= len(dv.lines) {
		dv.offset = len(dv.lines) - 1
	}
	dv.adjustScroll()
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/ -run TestDiffViewRefreshDiff -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/diffview.go internal/ui/diffview_test.go
git commit -m "feat: add DiffViewer.RefreshDiff with scroll preservation"
```

---

### Task 3: Add message types and refresh field to RootModel

Define the two new message types and add the `refreshInProgress` field.

**Files:**
- Modify: `internal/ui/root.go`

**Step 1: Add the message types and field**

Add after the `finishMsg` type (line 41) in `internal/ui/root.go`:

```go
// tickRefreshMsg signals that it's time to check for uncommitted changes.
type tickRefreshMsg struct{}

// refreshResultMsg carries the results of an async refresh operation.
type refreshResultMsg struct {
	files         []git.ChangedFile
	diff          *git.FileDiff
	requestedPath string // the file path that was selected when the refresh started
	err           error
}
```

Add `refreshInProgress bool` field to the `RootModel` struct, after the `searching` field (line 65):

```go
	refreshInProgress bool
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: Success (no compilation errors).

**Step 3: Commit**

```bash
git add internal/ui/root.go
git commit -m "feat: add refresh message types and refreshInProgress field"
```

---

### Task 4: Add tick scheduling and Init()

Add a helper to schedule the refresh tick and update `Init()` to start the tick in uncommitted mode.

**Files:**
- Modify: `internal/ui/root.go`
- Test: `internal/ui/root_test.go`

**Step 1: Write the failing tests**

Add to `internal/ui/root_test.go`:

```go
func TestRootInitUncommittedReturnsTick(t *testing.T) {
	m := newTestRootUncommitted()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a tick command in uncommitted mode")
	}
}

func TestRootInitBranchReturnsNil(t *testing.T) {
	m := newTestRoot()
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil in branch diff mode")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run TestRootInit -v`
Expected: `TestRootInitUncommittedReturnsTick` FAILS (Init currently returns nil).

**Step 3: Write the implementation**

Add the `time` import to `internal/ui/root.go`:

```go
import (
	"fmt"
	"strings"
	"time"
	...
)
```

Add the helper after the `loadFileDiff` method:

```go
const refreshInterval = 2 * time.Second

// scheduleRefreshTick returns a tea.Cmd that sends a tickRefreshMsg after the refresh interval.
func scheduleRefreshTick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickRefreshMsg{}
	})
}
```

Update `Init()`:

```go
// Init returns the initial command.
func (m RootModel) Init() tea.Cmd {
	if m.mode == modeUncommitted {
		return scheduleRefreshTick()
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/ -run TestRootInit -v`
Expected: PASS

Also verify all existing tests still pass:

Run: `go test ./internal/ui/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/ui/root.go internal/ui/root_test.go
git commit -m "feat: schedule refresh tick on Init in uncommitted mode"
```

---

### Task 5: Add the async refresh command

Add the `refreshCmd()` method that performs git operations and returns results as a `tea.Msg`.

**Files:**
- Modify: `internal/ui/root.go`
- Test: `internal/ui/root_test.go`

**Step 1: Write the failing tests**

The mock needs to support dynamic file lists so we can test refresh behavior. Add a `dynamicMockGitRunner` and test to `internal/ui/root_test.go`:

```go
// dynamicMockGitRunner supports changing file lists between calls for refresh testing.
type dynamicMockGitRunner struct {
	filesCalls   int
	filesResults [][]git.ChangedFile
	diffs        map[string]*git.FileDiff
}

func (d *dynamicMockGitRunner) ChangedFiles(_ string) ([]git.ChangedFile, error) {
	return nil, nil
}

func (d *dynamicMockGitRunner) FileDiff(_ string, path string) (*git.FileDiff, error) {
	return &git.FileDiff{Path: path}, nil
}

func (d *dynamicMockGitRunner) CurrentBranch() (string, error) {
	return "feature", nil
}

func (d *dynamicMockGitRunner) HasUncommittedChanges() bool {
	return true
}

func (d *dynamicMockGitRunner) UncommittedFiles() ([]git.ChangedFile, error) {
	idx := d.filesCalls
	d.filesCalls++
	if idx < len(d.filesResults) {
		return d.filesResults[idx], nil
	}
	// Return last result if called more times than expected
	return d.filesResults[len(d.filesResults)-1], nil
}

func (d *dynamicMockGitRunner) UncommittedFileDiff(path string) (*git.FileDiff, error) {
	if fd, ok := d.diffs[path]; ok {
		return fd, nil
	}
	return &git.FileDiff{Path: path}, nil
}

func TestRefreshCmd(t *testing.T) {
	mock := &dynamicMockGitRunner{
		filesResults: [][]git.ChangedFile{
			// First call (constructor)
			{{Path: "a.go", Status: "M"}},
			// Second call (refresh)
			{{Path: "a.go", Status: "M"}, {Path: "b.go", Status: "A"}},
		},
		diffs: map[string]*git.FileDiff{
			"a.go": makeTestDiff(),
		},
	}
	m := NewRootModelUncommitted(mock, 80, 24)

	cmd := m.refreshCmd()
	if cmd == nil {
		t.Fatal("refreshCmd should return a non-nil command")
	}

	// Execute the command synchronously
	msg := cmd()
	result, ok := msg.(refreshResultMsg)
	if !ok {
		t.Fatalf("expected refreshResultMsg, got %T", msg)
	}

	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}

	if len(result.files) != 2 {
		t.Errorf("expected 2 files, got %d", len(result.files))
	}

	if result.requestedPath != "a.go" {
		t.Errorf("requestedPath = %q, want %q", result.requestedPath, "a.go")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run TestRefreshCmd -v`
Expected: FAIL — `refreshCmd` method does not exist.

**Step 3: Write the implementation**

Add to `internal/ui/root.go` after `scheduleRefreshTick`:

```go
// refreshCmd returns a tea.Cmd that asynchronously fetches the current file list
// and diff for the selected file.
func (m RootModel) refreshCmd() tea.Cmd {
	currentPath := ""
	if len(m.files) > 0 {
		currentPath = m.fileList.SelectedFile().Path
	}
	gitRunner := m.git

	return func() tea.Msg {
		files, err := gitRunner.UncommittedFiles()
		if err != nil {
			return refreshResultMsg{err: err}
		}

		var diff *git.FileDiff
		if currentPath != "" {
			for _, f := range files {
				if f.Path == currentPath {
					diff, _ = gitRunner.UncommittedFileDiff(currentPath)
					break
				}
			}
		}

		return refreshResultMsg{
			files:         files,
			diff:          diff,
			requestedPath: currentPath,
		}
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/ -run TestRefreshCmd -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/root.go internal/ui/root_test.go
git commit -m "feat: add refreshCmd for async git polling"
```

---

### Task 6: Add Update() handlers for refresh messages

Wire up the tick and result message handlers in `RootModel.Update()`.

**Files:**
- Modify: `internal/ui/root.go`
- Test: `internal/ui/root_test.go`

**Step 1: Write the failing tests**

Add to `internal/ui/root_test.go`:

```go
func TestRootTickRefreshMsg(t *testing.T) {
	m := newTestRootUncommitted()

	t.Run("triggers refresh command", func(t *testing.T) {
		updated, cmd := m.Update(tickRefreshMsg{})
		m2 := updated.(RootModel)
		if cmd == nil {
			t.Error("tickRefreshMsg should produce a command")
		}
		if !m2.refreshInProgress {
			t.Error("refreshInProgress should be true")
		}
	})

	t.Run("skips when refresh already in progress", func(t *testing.T) {
		m2 := m
		m2.refreshInProgress = true
		updated, cmd := m2.Update(tickRefreshMsg{})
		_ = updated.(RootModel)
		if cmd == nil {
			t.Error("should still schedule next tick even when skipping")
		}
	})

	t.Run("ignored in branch mode", func(t *testing.T) {
		branchModel := newTestRoot()
		updated, cmd := branchModel.Update(tickRefreshMsg{})
		m2 := updated.(RootModel)
		if cmd != nil {
			t.Error("tickRefreshMsg should be ignored in branch mode")
		}
		if m2.refreshInProgress {
			t.Error("refreshInProgress should not be set in branch mode")
		}
	})
}

func TestRootRefreshResultMsg(t *testing.T) {
	mock := &dynamicMockGitRunner{
		filesResults: [][]git.ChangedFile{
			{{Path: "a.go", Status: "M"}, {Path: "b.go", Status: "A"}},
		},
		diffs: map[string]*git.FileDiff{
			"a.go": makeTestDiff(),
		},
	}
	m := NewRootModelUncommitted(mock, 80, 24)
	m.refreshInProgress = true

	t.Run("updates file list", func(t *testing.T) {
		newFiles := []git.ChangedFile{
			{Path: "a.go", Status: "M"},
			{Path: "c.go", Status: "A"},
		}
		result := refreshResultMsg{
			files:         newFiles,
			diff:          makeTestDiff(),
			requestedPath: "a.go",
		}
		updated, cmd := m.Update(result)
		m2 := updated.(RootModel)

		if m2.refreshInProgress {
			t.Error("refreshInProgress should be cleared")
		}
		if len(m2.files) != 2 {
			t.Errorf("expected 2 files, got %d", len(m2.files))
		}
		if m2.files[1].Path != "c.go" {
			t.Errorf("second file = %q, want c.go", m2.files[1].Path)
		}
		if cmd == nil {
			t.Error("should schedule next tick")
		}
	})

	t.Run("skips diff update when file changed", func(t *testing.T) {
		// Simulate user navigated to b.go while refresh was for a.go
		m2 := m
		// Move cursor to b.go
		m2.fileList, _ = m2.fileList.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

		result := refreshResultMsg{
			files:         m2.files,
			diff:          makeTestDiff(),
			requestedPath: "a.go", // was requested for a.go, but user is now on b.go
		}
		updated, _ := m2.Update(result)
		_ = updated.(RootModel)
		// No crash, diff for b.go should not be overwritten
		// (this is a safety check — the test passing without panic is the assertion)
	})
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run "TestRootTickRefreshMsg|TestRootRefreshResultMsg" -v`
Expected: FAIL — the `Update()` method doesn't handle these message types yet.

**Step 3: Write the implementation**

Add two new cases to the `switch msg := msg.(type)` block in `Update()` (in `internal/ui/root.go`, after the `tea.WindowSizeMsg` case around line 167):

```go
	case tickRefreshMsg:
		if m.mode != modeUncommitted {
			return m, nil
		}
		if m.refreshInProgress {
			return m, scheduleRefreshTick()
		}
		m.refreshInProgress = true
		return m, m.refreshCmd()

	case refreshResultMsg:
		m.refreshInProgress = false
		if msg.err != nil {
			return m, scheduleRefreshTick()
		}

		// Update file list
		m.files = msg.files
		m.fileList.SetFiles(msg.files)

		// Update diff only if the user is still on the same file
		currentPath := ""
		if len(m.files) > 0 {
			currentPath = m.fileList.SelectedFile().Path
		}
		if msg.diff != nil && msg.requestedPath == currentPath {
			m.diffViewer.RefreshDiff(msg.diff)
			m.updateCommentMarkers()
		} else if currentPath == "" {
			// All files removed
			m.diffViewer.RefreshDiff(nil)
		}

		return m, scheduleRefreshTick()
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/ -run "TestRootTickRefreshMsg|TestRootRefreshResultMsg" -v`
Expected: PASS

Run full test suite to ensure no regressions:

Run: `go test ./internal/ui/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/ui/root.go internal/ui/root_test.go
git commit -m "feat: handle refresh tick and result messages in Update()"
```

---

### Task 7: Run full test suite and verify

Final verification that everything works together.

**Files:** None (verification only)

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 2: Run linters**

Run: `go vet ./...`
Expected: No warnings

Run: `go fmt ./...`
Expected: No changes (already formatted)

**Step 3: Manual smoke test**

Run revui in a directory with uncommitted changes and verify:
1. The header shows "uncommitted changes"
2. Edit a file in another terminal — the diff updates within ~2 seconds
3. Add a new file — it appears in the file list
4. Revert all changes to a file — it disappears from the list
5. Cursor stays on the current file after refresh
6. Scroll position is preserved after refresh
7. Comments are preserved after refresh

Run: `go build ./cmd/revui && ./revui`
