package ui

import (
	"fmt"
	"strings"
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
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(RootModel)
	if m.focus != focusDiffViewer {
		t.Errorf("after l: focus = %d, want focusDiffViewer", m.focus)
	}

	// h switches back to file list
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(RootModel)
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

func TestRootHelpToggle(t *testing.T) {
	m := newTestRoot()

	// ? shows help
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(RootModel)
	if !m.showHelp {
		t.Error("? should show help")
	}

	// ? again hides help
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(RootModel)
	if m.showHelp {
		t.Error("second ? should hide help")
	}
}

func newTestRootUncommitted() RootModel {
	mock := &mockGitRunner{
		files: []git.ChangedFile{
			{Path: "main.go", Status: "M"},
			{Path: "newfile.go", Status: "A"},
			{Path: "image.png", Status: "B"},
		},
		diffs: map[string]*git.FileDiff{
			"main.go":   makeTestDiff(),
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
}

func TestRootUncommittedFileList(t *testing.T) {
	m := newTestRootUncommitted()
	if len(m.files) != 3 {
		t.Errorf("expected 3 files, got %d", len(m.files))
	}
}

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

func TestRootBinaryFileComment(t *testing.T) {
	m := newTestRootUncommitted()

	// Navigate to binary file (3rd file, index 2)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(RootModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(RootModel)

	// Enter diff viewer
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(RootModel)
	if m.focus != focusDiffViewer {
		t.Fatal("expected focus on diff viewer")
	}

	// Press c to comment on binary file
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(RootModel)
	if m.focus != focusCommentInput {
		t.Error("expected comment input to activate on binary file")
	}
}

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

func TestRefreshCmdEmptyFileList(t *testing.T) {
	mock := &dynamicMockGitRunner{
		filesResults: [][]git.ChangedFile{
			{}, // constructor gets empty list
			{{Path: "a.go", Status: "A"}}, // refresh finds a new file
		},
		diffs: map[string]*git.FileDiff{},
	}
	m := NewRootModelUncommitted(mock, 80, 24)

	cmd := m.refreshCmd()
	msg := cmd()
	result := msg.(refreshResultMsg)

	if result.requestedPath != "" {
		t.Errorf("requestedPath = %q, want empty", result.requestedPath)
	}
	if result.diff != nil {
		t.Error("diff should be nil when no file was selected")
	}
	if len(result.files) != 1 {
		t.Errorf("expected 1 file, got %d", len(result.files))
	}
}

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
	})

	t.Run("handles error gracefully", func(t *testing.T) {
		m2 := m
		result := refreshResultMsg{err: fmt.Errorf("git error")}
		updated, cmd := m2.Update(result)
		m3 := updated.(RootModel)
		if m3.refreshInProgress {
			t.Error("refreshInProgress should be cleared on error")
		}
		if cmd == nil {
			t.Error("should reschedule tick after error")
		}
		// File list should remain unchanged
		if len(m3.files) != len(m.files) {
			t.Error("file list should not change on error")
		}
	})

	t.Run("handles all files removed", func(t *testing.T) {
		m2 := m
		result := refreshResultMsg{
			files:         nil,
			requestedPath: "a.go",
		}
		updated, cmd := m2.Update(result)
		m3 := updated.(RootModel)
		if len(m3.files) != 0 {
			t.Errorf("expected 0 files, got %d", len(m3.files))
		}
		if cmd == nil {
			t.Error("should reschedule tick")
		}
	})
}
