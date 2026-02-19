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
		// b.go removed; cursor stays at index 1 (now c.go, the nearest neighbor)
		if fl2.cursor != 1 {
			t.Errorf("cursor = %d, want 1 (neighbor of removed file)", fl2.cursor)
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
