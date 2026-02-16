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
