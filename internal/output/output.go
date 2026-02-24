package output

import "strings"

// TargetKind identifies the type of output destination.
type TargetKind int

const (
	TargetClaude TargetKind = iota
	TargetTmuxBuffer
	TargetClipboard
	TargetFile
)

// OutputTarget represents a destination for review output.
type OutputTarget struct {
	Kind       TargetKind
	Label      string
	TmuxTarget string // pane identifier for tmux send-keys (Claude targets only)
}

// parseTmuxPanes parses output from `tmux list-panes -a -F '#{session_name}:#{window_index}.#{pane_index} #{pane_current_command} #{pane_pid}'`.
// Returns a slice of OutputTarget for each pane running claude.
// currentPane is the $TMUX_PANE value, currently unused but reserved for future self-exclusion.
func parseTmuxPanes(output, currentPane string) []OutputTarget {
	lines := strings.Split(output, "\n")
	var targets []OutputTarget

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			// Skip malformed lines.
			continue
		}

		paneID := fields[0]
		command := fields[1]

		if command == "claude" {
			targets = append(targets, OutputTarget{
				Kind:       TargetClaude,
				Label:      paneID + "  " + command,
				TmuxTarget: paneID,
			})
		}
	}

	return targets
}
