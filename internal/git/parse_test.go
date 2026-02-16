package git

import (
	"os"
	"testing"
)

func TestParseFileDiff(t *testing.T) {
	raw, err := os.ReadFile("testdata/simple.diff")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	diffs, err := ParseDiff(string(raw))
	if err != nil {
		t.Fatalf("ParseDiff: %v", err)
	}

	if len(diffs) != 1 {
		t.Fatalf("got %d file diffs, want 1", len(diffs))
	}

	fd := diffs[0]
	if fd.Path != "main.go" {
		t.Errorf("Path = %q, want %q", fd.Path, "main.go")
	}

	if len(fd.Hunks) != 1 {
		t.Fatalf("got %d hunks, want 1", len(fd.Hunks))
	}

	h := fd.Hunks[0]
	if h.OldStart != 1 || h.OldCount != 5 {
		t.Errorf("old range = %d,%d, want 1,5", h.OldStart, h.OldCount)
	}
	if h.NewStart != 1 || h.NewCount != 6 {
		t.Errorf("new range = %d,%d, want 1,6", h.NewStart, h.NewCount)
	}

	var added, removed, context int
	for _, line := range h.Lines {
		switch line.Type {
		case LineAdded:
			added++
		case LineRemoved:
			removed++
		case LineContext:
			context++
		}
	}
	if added != 2 {
		t.Errorf("added lines = %d, want 2", added)
	}
	if removed != 1 {
		t.Errorf("removed lines = %d, want 1", removed)
	}
	// 4 context lines: "package main", blank line, "func main() {", "}"
	if context != 4 {
		t.Errorf("context lines = %d, want 4", context)
	}
}

func TestParseLineNumbers(t *testing.T) {
	raw, err := os.ReadFile("testdata/simple.diff")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	diffs, err := ParseDiff(string(raw))
	if err != nil {
		t.Fatalf("ParseDiff: %v", err)
	}

	lines := diffs[0].Hunks[0].Lines

	// Verify context lines have both old and new line numbers.
	for _, line := range lines {
		if line.Type == LineContext {
			if line.OldLineNo == 0 {
				t.Errorf("context line %q: OldLineNo should not be 0", line.Content)
			}
			if line.NewLineNo == 0 {
				t.Errorf("context line %q: NewLineNo should not be 0", line.Content)
			}
		}
	}

	// Verify removed lines have only old line numbers.
	for _, line := range lines {
		if line.Type == LineRemoved {
			if line.OldLineNo == 0 {
				t.Errorf("removed line %q: OldLineNo should not be 0", line.Content)
			}
			if line.NewLineNo != 0 {
				t.Errorf("removed line %q: NewLineNo = %d, want 0", line.Content, line.NewLineNo)
			}
		}
	}

	// Verify added lines have only new line numbers.
	for _, line := range lines {
		if line.Type == LineAdded {
			if line.NewLineNo == 0 {
				t.Errorf("added line %q: NewLineNo should not be 0", line.Content)
			}
			if line.OldLineNo != 0 {
				t.Errorf("added line %q: OldLineNo = %d, want 0", line.Content, line.OldLineNo)
			}
		}
	}

	// Verify specific line numbers.
	// Hunk starts at old=1, new=1.
	// Line 0: " package main"      -> context: old=1, new=1
	// Line 1: ""                    -> context: old=2, new=2 (blank line)
	// Line 2: " func main() {"     -> context: old=3, new=3
	// Line 3: "-\tfmt.Println..."   -> removed: old=4, new=0
	// Line 4: "+\tfmt.Println..."   -> added:   old=0, new=4
	// Line 5: "+\tfmt.Println..."   -> added:   old=0, new=5
	// Line 6: " }"                  -> context: old=5, new=6

	expected := []struct {
		oldNo int
		newNo int
		typ   LineType
	}{
		{1, 1, LineContext},
		{2, 2, LineContext},
		{3, 3, LineContext},
		{4, 0, LineRemoved},
		{0, 4, LineAdded},
		{0, 5, LineAdded},
		{5, 6, LineContext},
	}

	if len(lines) != len(expected) {
		t.Fatalf("got %d lines, want %d", len(lines), len(expected))
	}

	for i, want := range expected {
		got := lines[i]
		if got.OldLineNo != want.oldNo {
			t.Errorf("line %d: OldLineNo = %d, want %d", i, got.OldLineNo, want.oldNo)
		}
		if got.NewLineNo != want.newNo {
			t.Errorf("line %d: NewLineNo = %d, want %d", i, got.NewLineNo, want.newNo)
		}
		if got.Type != want.typ {
			t.Errorf("line %d: Type = %v, want %v", i, got.Type, want.typ)
		}
	}
}

func TestParseNameStatus(t *testing.T) {
	raw := "M\tmain.go\nA\tnew.go"

	files := ParseNameStatus(raw)

	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}

	if files[0].Path != "main.go" || files[0].Status != "M" {
		t.Errorf("files[0] = {%q, %q}, want {%q, %q}", files[0].Path, files[0].Status, "main.go", "M")
	}
	if files[1].Path != "new.go" || files[1].Status != "A" {
		t.Errorf("files[1] = {%q, %q}, want {%q, %q}", files[1].Path, files[1].Status, "new.go", "A")
	}
}

func TestParseNameStatusEmpty(t *testing.T) {
	files := ParseNameStatus("")
	if len(files) != 0 {
		t.Errorf("got %d files from empty input, want 0", len(files))
	}
}

func TestParseDiffEmpty(t *testing.T) {
	diffs, err := ParseDiff("")
	if err != nil {
		t.Fatalf("ParseDiff on empty input: %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("got %d diffs from empty input, want 0", len(diffs))
	}
}
