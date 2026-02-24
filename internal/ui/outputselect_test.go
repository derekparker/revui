package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/deparker/revui/internal/output"
)

func testTargets() []output.OutputTarget {
	return []output.OutputTarget{
		{Kind: output.TargetClaude, Label: "revui:0.0  claude", TmuxTarget: "revui:0.0"},
		{Kind: output.TargetClaude, Label: "go:0.0  claude", TmuxTarget: "go:0.0"},
		{Kind: output.TargetTmuxBuffer, Label: "tmux paste buffer"},
		{Kind: output.TargetClipboard, Label: "System clipboard"},
		{Kind: output.TargetFile, Label: "Write to file"},
	}
}

func TestOutputSelector_Navigation(t *testing.T) {
	targets := testTargets()
	os := NewOutputSelector(targets, 80, 24)

	// j moves cursor down
	os, _ = os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if os.cursor != 1 {
		t.Errorf("Expected cursor 1 after j, got %d", os.cursor)
	}

	// k moves cursor up
	os, _ = os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if os.cursor != 0 {
		t.Errorf("Expected cursor 0 after k, got %d", os.cursor)
	}

	// k at 0 stays at 0
	os, _ = os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if os.cursor != 0 {
		t.Errorf("Expected cursor to stay at 0, got %d", os.cursor)
	}

	// Move to last position
	for i := 0; i < len(targets); i++ {
		os, _ = os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	// j at last stays at last
	lastIdx := len(targets) - 1
	if os.cursor != lastIdx {
		t.Errorf("Expected cursor at %d, got %d", lastIdx, os.cursor)
	}

	os, _ = os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if os.cursor != lastIdx {
		t.Errorf("Expected cursor to stay at %d after j, got %d", lastIdx, os.cursor)
	}
}

func TestOutputSelector_NavigationArrowKeys(t *testing.T) {
	targets := testTargets()
	os := NewOutputSelector(targets, 80, 24)

	// Down arrow moves cursor down
	os, _ = os.Update(tea.KeyMsg{Type: tea.KeyDown})
	if os.cursor != 1 {
		t.Errorf("Expected cursor 1 after down arrow, got %d", os.cursor)
	}

	// Up arrow moves cursor up
	os, _ = os.Update(tea.KeyMsg{Type: tea.KeyUp})
	if os.cursor != 0 {
		t.Errorf("Expected cursor 0 after up arrow, got %d", os.cursor)
	}
}

func TestOutputSelector_Select(t *testing.T) {
	targets := testTargets()
	os := NewOutputSelector(targets, 80, 24)

	// Move to second item
	os, _ = os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Enter sends OutputSelectMsg with correct target
	_, cmd := os.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Expected command after Enter, got nil")
	}

	msg := cmd()
	selectMsg, ok := msg.(OutputSelectMsg)
	if !ok {
		t.Fatalf("Expected OutputSelectMsg, got %T", msg)
	}

	if selectMsg.Target.Label != targets[1].Label {
		t.Errorf("Expected target %q, got %q", targets[1].Label, selectMsg.Target.Label)
	}
}

func TestOutputSelector_Cancel(t *testing.T) {
	targets := testTargets()
	os := NewOutputSelector(targets, 80, 24)

	// q sends OutputCancelMsg
	_, cmd := os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("Expected command after q, got nil")
	}

	msg := cmd()
	_, ok := msg.(OutputCancelMsg)
	if !ok {
		t.Fatalf("Expected OutputCancelMsg, got %T", msg)
	}
}

func TestOutputSelector_EscCancel(t *testing.T) {
	targets := testTargets()
	os := NewOutputSelector(targets, 80, 24)

	// esc also sends OutputCancelMsg
	_, cmd := os.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("Expected command after esc, got nil")
	}

	msg := cmd()
	_, ok := msg.(OutputCancelMsg)
	if !ok {
		t.Fatalf("Expected OutputCancelMsg, got %T", msg)
	}
}

func TestOutputSelector_ViewNotEmpty(t *testing.T) {
	targets := testTargets()
	os := NewOutputSelector(targets, 80, 24)

	view := os.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}
}

func TestOutputSelector_ViewContainsTitle(t *testing.T) {
	targets := testTargets()
	os := NewOutputSelector(targets, 80, 24)

	view := os.View()
	if !strings.Contains(view, "Send review to:") {
		t.Error("Expected view to contain title 'Send review to:'")
	}
}

func TestOutputSelector_ViewContainsSeparator(t *testing.T) {
	targets := testTargets()
	os := NewOutputSelector(targets, 80, 24)

	view := os.View()
	// When Claude targets present, should contain separator
	if !strings.Contains(view, "── or ──") {
		t.Error("Expected view to contain separator '── or ──' when Claude targets present")
	}
}

func TestOutputSelector_NoTargets(t *testing.T) {
	os := NewOutputSelector(nil, 80, 24)

	view := os.View()
	if view == "" {
		t.Error("Expected non-empty view even with no targets")
	}

	if !strings.Contains(view, "No output targets available") {
		t.Error("Expected 'No output targets available' message")
	}
}

func TestOutputSelector_ErrorDisplay(t *testing.T) {
	targets := testTargets()
	os := NewOutputSelector(targets, 80, 24)

	os.SetError("failed to deliver")

	view := os.View()
	if !strings.Contains(view, "Error: failed to deliver") {
		t.Error("Expected view to contain error message")
	}
}
