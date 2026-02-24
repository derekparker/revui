package output

import (
	"os"
	"strings"
	"testing"
)

func TestParseTmuxPanes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		currentPane string
		want        []OutputTarget
	}{
		{
			name: "multiple panes some claude",
			input: `revui:0.0 zsh 12345
revui:0.1 claude 12346
revui:1.0 nvim 12347
revui:1.1 claude 12348`,
			currentPane: "revui:0.0",
			want: []OutputTarget{
				{
					Kind:       TargetClaude,
					Label:      "revui:0.1  claude",
					TmuxTarget: "revui:0.1",
				},
				{
					Kind:       TargetClaude,
					Label:      "revui:1.1  claude",
					TmuxTarget: "revui:1.1",
				},
			},
		},
		{
			name:        "empty input",
			input:       "",
			currentPane: "revui:0.0",
			want:        []OutputTarget{},
		},
		{
			name: "no claude panes",
			input: `session:0.0 zsh 12345
session:0.1 nvim 12346`,
			currentPane: "session:0.0",
			want:        []OutputTarget{},
		},
		{
			name:        "single claude pane",
			input:       "mysession:2.3 claude 99999",
			currentPane: "mysession:0.0",
			want: []OutputTarget{
				{
					Kind:       TargetClaude,
					Label:      "mysession:2.3  claude",
					TmuxTarget: "mysession:2.3",
				},
			},
		},
		{
			name: "lines with unexpected format",
			input: `good:0.0 claude 12345
malformed
also-bad
another:1.0 claude 67890`,
			currentPane: "good:0.0",
			want: []OutputTarget{
				{
					Kind:       TargetClaude,
					Label:      "good:0.0  claude",
					TmuxTarget: "good:0.0",
				},
				{
					Kind:       TargetClaude,
					Label:      "another:1.0  claude",
					TmuxTarget: "another:1.0",
				},
			},
		},
		{
			name: "whitespace variations",
			input: `

session:0.0 claude 12345

session:1.0 zsh 12346

`,
			currentPane: "session:0.0",
			want: []OutputTarget{
				{
					Kind:       TargetClaude,
					Label:      "session:0.0  claude",
					TmuxTarget: "session:0.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTmuxPanes(tt.input, tt.currentPane)

			if len(got) != len(tt.want) {
				t.Fatalf("got %d targets, want %d", len(got), len(tt.want))
			}

			for i := range got {
				if got[i].Kind != tt.want[i].Kind {
					t.Errorf("target[%d].Kind = %v, want %v", i, got[i].Kind, tt.want[i].Kind)
				}
				if got[i].Label != tt.want[i].Label {
					t.Errorf("target[%d].Label = %q, want %q", i, got[i].Label, tt.want[i].Label)
				}
				if got[i].TmuxTarget != tt.want[i].TmuxTarget {
					t.Errorf("target[%d].TmuxTarget = %q, want %q", i, got[i].TmuxTarget, tt.want[i].TmuxTarget)
				}
			}
		})
	}
}

func TestTargetKindConstants(t *testing.T) {
	// Verify that target kinds are distinct.
	kinds := []TargetKind{
		TargetClaude,
		TargetTmuxBuffer,
		TargetClipboard,
		TargetFile,
	}

	seen := make(map[TargetKind]bool)
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("duplicate TargetKind value: %v", k)
		}
		seen[k] = true
	}
}

func TestDetectTargets(t *testing.T) {
	tests := []struct {
		name     string
		tmuxEnv  string
		tmuxPane string
		want     []OutputTarget
	}{
		{
			name:     "not in tmux",
			tmuxEnv:  "",
			tmuxPane: "",
			want: []OutputTarget{
				{Kind: TargetClipboard, Label: "System clipboard"},
				{Kind: TargetFile, Label: "Write to file"},
			},
		},
		{
			name:     "in tmux but empty pane list",
			tmuxEnv:  "/tmp/tmux-1000/default,12345,0",
			tmuxPane: "session:0.0",
			want: []OutputTarget{
				{Kind: TargetTmuxBuffer, Label: "tmux paste buffer"},
				{Kind: TargetClipboard, Label: "System clipboard"},
				{Kind: TargetFile, Label: "Write to file"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectTargets(tt.tmuxEnv, tt.tmuxPane)

			// For the "not in tmux" case, verify we get exactly clipboard + file
			if tt.tmuxEnv == "" {
				if len(got) != 2 {
					t.Fatalf("got %d targets, want 2", len(got))
				}
			}

			// Verify clipboard and file are always the last two items
			if len(got) < 2 {
				t.Fatalf("got %d targets, want at least 2", len(got))
			}

			lastTwo := got[len(got)-2:]
			if lastTwo[0].Kind != TargetClipboard || lastTwo[0].Label != "System clipboard" {
				t.Errorf("second-to-last target = {Kind: %v, Label: %q}, want {Kind: TargetClipboard, Label: \"System clipboard\"}", lastTwo[0].Kind, lastTwo[0].Label)
			}
			if lastTwo[1].Kind != TargetFile || lastTwo[1].Label != "Write to file" {
				t.Errorf("last target = {Kind: %v, Label: %q}, want {Kind: TargetFile, Label: \"Write to file\"}", lastTwo[1].Kind, lastTwo[1].Label)
			}
		})
	}
}

func TestDeliverFile(t *testing.T) {
	content := "# Code Review\n\nTest content"
	target := OutputTarget{
		Kind:  TargetFile,
		Label: "Write to file",
	}

	msg, err := Deliver(target, content)
	if err != nil {
		t.Fatalf("Deliver failed: %v", err)
	}

	// Check that message contains the path
	if !strings.Contains(msg, "Review written to") {
		t.Errorf("message %q does not contain expected prefix", msg)
	}
	if !strings.Contains(msg, "/tmp/revui-review-") {
		t.Errorf("message %q does not contain expected path", msg)
	}

	// Extract the file path from the message
	// Message format: "Review written to <path>"
	parts := strings.Split(msg, "Review written to ")
	if len(parts) != 2 {
		t.Fatalf("unexpected message format: %q", msg)
	}
	filePath := strings.TrimSpace(parts[1])

	// Verify the file exists and has correct content
	gotContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if string(gotContent) != content {
		t.Errorf("file content = %q, want %q", string(gotContent), content)
	}

	// Clean up
	os.Remove(filePath)
}
