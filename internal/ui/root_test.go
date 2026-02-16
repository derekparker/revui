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
