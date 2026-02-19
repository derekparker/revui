package ui

import (
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
