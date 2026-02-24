package output

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

// DetectTargets discovers available output destinations.
// tmuxEnv is the value of $TMUX (empty if not in tmux).
// tmuxPane is the value of $TMUX_PANE.
func DetectTargets(tmuxEnv, tmuxPane string) []OutputTarget {
	var targets []OutputTarget

	if tmuxEnv != "" {
		// Try to list tmux panes
		cmd := exec.Command("tmux", "list-panes", "-a", "-F", "#{session_name}:#{window_index}.#{pane_index} #{pane_current_command} #{pane_pid}")
		output, err := cmd.Output()
		if err == nil {
			claudeTargets := parseTmuxPanes(string(output), tmuxPane)
			targets = append(targets, claudeTargets...)
		}

		// Add tmux paste buffer option
		targets = append(targets, OutputTarget{
			Kind:  TargetTmuxBuffer,
			Label: "tmux paste buffer",
		})
	}

	// Always add clipboard and file options
	targets = append(targets, OutputTarget{
		Kind:  TargetClipboard,
		Label: "System clipboard",
	})
	targets = append(targets, OutputTarget{
		Kind:  TargetFile,
		Label: "Write to file",
	})

	return targets
}

// Deliver sends the review content to the specified target.
// Returns a human-readable status message on success.
func Deliver(target OutputTarget, content string) (string, error) {
	switch target.Kind {
	case TargetClaude:
		return deliverToClaude(target, content)
	case TargetTmuxBuffer:
		return deliverToTmuxBuffer(content)
	case TargetClipboard:
		return deliverToClipboard(content)
	case TargetFile:
		return deliverToFile(content)
	default:
		return "", fmt.Errorf("unknown target kind: %v", target.Kind)
	}
}

// reviewFilePath generates a timestamped file path for review output.
func reviewFilePath() string {
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("revui-review-%d.md", timestamp)
	return filepath.Join("/tmp", filename)
}

// deliverToClaude writes content to a temp file and sends an @path reference to the Claude pane.
func deliverToClaude(target OutputTarget, content string) (string, error) {
	path := reviewFilePath()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write review file: %w", err)
	}

	// Send @path reference to Claude pane (without pressing Enter)
	atRef := fmt.Sprintf("@%s ", path)
	cmd := exec.Command("tmux", "send-keys", "-t", target.TmuxTarget, atRef)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to send to tmux pane: %w", err)
	}

	return fmt.Sprintf("Review sent to Claude at %s (file: %s)", target.TmuxTarget, path), nil
}

// deliverToTmuxBuffer loads content into the tmux paste buffer.
func deliverToTmuxBuffer(content string) (string, error) {
	cmd := exec.Command("tmux", "load-buffer", "-")
	cmd.Stdin = strings.NewReader(content)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to load tmux buffer: %w", err)
	}

	return "Review loaded into tmux paste buffer. Use prefix + ] to paste.", nil
}

// deliverToClipboard tries available clipboard utilities to copy content.
func deliverToClipboard(content string) (string, error) {
	// Try clipboard utilities in order
	utilities := [][]string{
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--input", "--clipboard"},
		{"wl-copy"},
	}

	var lastErr error
	for _, util := range utilities {
		cmd := exec.Command(util[0], util[1:]...)
		cmd.Stdin = strings.NewReader(content)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			lastErr = err
			continue
		}

		// Success
		return "Review comments copied to clipboard.", nil
	}

	if lastErr != nil {
		return "", fmt.Errorf("no clipboard utility available (tried xclip, xsel, wl-copy): %w", lastErr)
	}

	return "", fmt.Errorf("no clipboard utility available (tried xclip, xsel, wl-copy)")
}

// deliverToFile writes content to a timestamped file.
func deliverToFile(content string) (string, error) {
	path := reviewFilePath()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write review file: %w", err)
	}

	return fmt.Sprintf("Review written to %s", path), nil
}
