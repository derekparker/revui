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
	if line != nil {
		// cursor 0 is on the hunk header, so line should be nil
		t.Error("expected nil line on hunk header")
	}

	// Move to first code line
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	line = dv.CurrentLine()
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
