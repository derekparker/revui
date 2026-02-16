package ui

import (
	"testing"

	"github.com/deparker/revui/internal/git"
)

func TestSideBySideRender(t *testing.T) {
	lines := []git.Line{
		{Content: "old line", Type: git.LineRemoved, OldLineNo: 1},
		{Content: "new line", Type: git.LineAdded, NewLineNo: 1},
		{Content: "context", Type: git.LineContext, OldLineNo: 2, NewLineNo: 2},
	}

	pairs := BuildSideBySidePairs(lines)
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}

	// First pair: removed + added
	if pairs[0].Left == nil || pairs[0].Right == nil {
		t.Error("first pair should have both sides")
	}

	// Second pair: context on both sides
	if pairs[1].Left == nil || pairs[1].Right == nil {
		t.Error("second pair should have both sides for context")
	}
}

func TestSideBySideOnlyAdded(t *testing.T) {
	lines := []git.Line{
		{Content: "new line 1", Type: git.LineAdded, NewLineNo: 1},
		{Content: "new line 2", Type: git.LineAdded, NewLineNo: 2},
	}

	pairs := BuildSideBySidePairs(lines)
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}

	for i, p := range pairs {
		if p.Left != nil {
			t.Errorf("pair %d: expected nil Left for added-only line", i)
		}
		if p.Right == nil {
			t.Errorf("pair %d: expected non-nil Right", i)
		}
	}
}

func TestSideBySideOnlyRemoved(t *testing.T) {
	lines := []git.Line{
		{Content: "old line", Type: git.LineRemoved, OldLineNo: 1},
		{Content: "context", Type: git.LineContext, OldLineNo: 2, NewLineNo: 1},
	}

	pairs := BuildSideBySidePairs(lines)
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}

	// First pair: removed only (left side)
	if pairs[0].Left == nil {
		t.Error("first pair should have Left side for removed line")
	}
	if pairs[0].Right != nil {
		t.Error("first pair should have nil Right for removed-only line")
	}
}
