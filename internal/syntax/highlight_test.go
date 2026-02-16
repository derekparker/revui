package syntax

import "testing"

func TestHighlightGoLine(t *testing.T) {
	h := NewHighlighter()
	result := h.HighlightLine("main.go", "func hello() {")
	if result == "" {
		t.Error("expected non-empty highlighted output")
	}
	if result == "func hello() {" {
		t.Error("expected ANSI-styled output, got plain text")
	}
}

func TestHighlightUnknownExtension(t *testing.T) {
	h := NewHighlighter()
	result := h.HighlightLine("unknown.xyz", "some content")
	if result == "" {
		t.Error("expected non-empty output even for unknown extension")
	}
}

func TestHighlightEmptyLine(t *testing.T) {
	h := NewHighlighter()
	result := h.HighlightLine("main.go", "")
	if len(result) > 20 {
		t.Errorf("expected minimal output for empty line, got len=%d", len(result))
	}
}
