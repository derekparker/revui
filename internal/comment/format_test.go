package comment

import (
	"strings"
	"testing"

	"github.com/deparker/revui/internal/git"
)

func TestFormatSingleComment(t *testing.T) {
	store := NewStore()
	store.Add(Comment{
		FilePath:  "main.go",
		StartLine: 10,
		EndLine:   10,
		LineType:  git.LineAdded,
		Body:      "This needs error handling.",
	})

	out := Format(store.All())

	expected := "main.go\n- L10 (added): This needs error handling.\n"
	if out != expected {
		t.Errorf("got:\n%s\nwant:\n%s", out, expected)
	}
}

func TestFormatRangeComment(t *testing.T) {
	store := NewStore()
	store.Add(Comment{
		FilePath:  "util.go",
		StartLine: 5,
		EndLine:   8,
		LineType:  git.LineRemoved,
		Body:      "Why was this removed?",
	})

	out := Format(store.All())

	expected := "util.go\n- L5-8 (removed): Why was this removed?\n"
	if out != expected {
		t.Errorf("got:\n%s\nwant:\n%s", out, expected)
	}
}

func TestFormatGroupsByFile(t *testing.T) {
	store := NewStore()
	store.Add(Comment{FilePath: "a.go", StartLine: 1, EndLine: 1, Body: "first"})
	store.Add(Comment{FilePath: "b.go", StartLine: 1, EndLine: 1, Body: "second"})
	store.Add(Comment{FilePath: "a.go", StartLine: 5, EndLine: 5, Body: "third"})

	out := Format(store.All())

	if strings.Count(out, "a.go\n") != 1 {
		t.Error("a.go header should appear exactly once")
	}
	if strings.Count(out, "b.go\n") != 1 {
		t.Error("b.go header should appear exactly once")
	}
}

func TestStoreAddAndDelete(t *testing.T) {
	store := NewStore()
	store.Add(Comment{FilePath: "a.go", StartLine: 1, EndLine: 1, Body: "hello"})
	store.Add(Comment{FilePath: "a.go", StartLine: 5, EndLine: 5, Body: "world"})

	if len(store.All()) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(store.All()))
	}

	store.Delete("a.go", 1)
	if len(store.All()) != 1 {
		t.Fatalf("expected 1 comment after delete, got %d", len(store.All()))
	}

	c := store.Get("a.go", 5)
	if c == nil || c.Body != "world" {
		t.Error("wrong comment returned after delete")
	}
}

func TestStoreGetByFileLine(t *testing.T) {
	store := NewStore()
	store.Add(Comment{FilePath: "a.go", StartLine: 10, EndLine: 10, Body: "found"})

	c := store.Get("a.go", 10)
	if c == nil {
		t.Fatal("expected comment, got nil")
	}
	if c.Body != "found" {
		t.Errorf("body = %q, want %q", c.Body, "found")
	}

	c = store.Get("a.go", 11)
	if c != nil {
		t.Error("expected nil for non-existent line")
	}
}

func BenchmarkFormat(b *testing.B) {
	comments := []Comment{
		{FilePath: "a.go", StartLine: 1, EndLine: 1, LineType: git.LineAdded, Body: "first comment"},
		{FilePath: "a.go", StartLine: 10, EndLine: 15, LineType: git.LineRemoved, Body: "second comment"},
		{FilePath: "b.go", StartLine: 5, EndLine: 5, LineType: git.LineContext, Body: "third comment"},
		{FilePath: "c.go", StartLine: 20, EndLine: 20, LineType: git.LineAdded, Body: "fourth comment"},
	}
	b.ResetTimer()
	for b.Loop() {
		Format(comments)
	}
}
