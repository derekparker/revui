package git

import "testing"

func TestLineTypeString(t *testing.T) {
	tests := []struct {
		lt   LineType
		want string
	}{
		{LineAdded, "added"},
		{LineRemoved, "removed"},
		{LineContext, "context"},
	}
	for _, tt := range tests {
		if got := tt.lt.String(); got != tt.want {
			t.Errorf("LineType(%d).String() = %q, want %q", tt.lt, got, tt.want)
		}
	}
}

func TestFileStatusString(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"A", "added"},
		{"M", "modified"},
		{"D", "deleted"},
		{"R", "renamed"},
		{"X", "unknown"},
	}
	for _, tt := range tests {
		if got := FileStatusString(tt.status); got != tt.want {
			t.Errorf("FileStatusString(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}
