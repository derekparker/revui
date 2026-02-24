package output

import (
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
