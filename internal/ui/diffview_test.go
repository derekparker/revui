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
	// Check that the diff content appears in the rendered view
	if !strings.Contains(view, "package") {
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

func TestDiffViewCommentNavigation(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())
	dv.SetCommentLines(map[int]bool{1: true, 4: true})

	// ]c jumps to next comment
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if dv.CursorLine() != 1 {
		t.Errorf("after ]c: cursor = %d, want 1", dv.CursorLine())
	}

	// ]c again jumps to index 4
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if dv.CursorLine() != 4 {
		t.Errorf("after second ]c: cursor = %d, want 4", dv.CursorLine())
	}

	// [c jumps back to index 1
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if dv.CursorLine() != 1 {
		t.Errorf("after [c: cursor = %d, want 1", dv.CursorLine())
	}
}

func TestDiffViewSearch(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())

	dv.SetSearch("new line")
	matches := dv.SearchMatches()
	if len(matches) == 0 {
		t.Error("expected search matches")
	}
}

func TestDiffViewSideBySideToggle(t *testing.T) {
	dv := NewDiffViewer(80, 20)
	dv.SetDiff(makeTestDiff())

	if dv.IsSideBySide() {
		t.Error("should not be side-by-side initially")
	}

	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyTab})
	if !dv.IsSideBySide() {
		t.Error("should be side-by-side after Tab")
	}

	dv, _ = dv.Update(tea.KeyMsg{Type: tea.KeyTab})
	if dv.IsSideBySide() {
		t.Error("should not be side-by-side after second Tab")
	}
}
