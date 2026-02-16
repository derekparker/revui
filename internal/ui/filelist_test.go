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
