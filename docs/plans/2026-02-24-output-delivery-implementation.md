# Output Delivery Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace clipboard-only output with an in-TUI selection screen that detects Claude Code instances in tmux and offers multiple delivery targets (Claude pane, tmux buffer, system clipboard, file).

**Architecture:** New `OutputSelector` sub-model in `internal/ui/` with focus routing through `RootModel`. New `internal/output/` package for delivery logic (tmux detection, file writing, send-keys). The `OutputSelector` follows the same `Init/Update/View` pattern as `FileList` and `CommentInput`.

**Tech Stack:** Go, Bubble Tea, lipgloss, `os/exec` for tmux commands, `atotto/clipboard` for system clipboard fallback.

---

### Task 1: Create the output package types and detection logic

**Files:**
- Create: `internal/output/output.go`
- Create: `internal/output/output_test.go`

**Step 1: Write the failing test for tmux pane parsing**

```go
package output

import (
	"testing"
)

func TestParseTmuxPanes(t *testing.T) {
	input := "revui:0.0 claude 12345\nrevui:1.0 lazygit 12346\ngo:0.0 claude 12347\n"
	currentPane := "%5" // simulated TMUX_PANE

	targets := parseTmuxPanes(input, currentPane)

	if len(targets) != 2 {
		t.Fatalf("expected 2 claude targets, got %d", len(targets))
	}
	if targets[0].TmuxTarget != "revui:0.0" {
		t.Errorf("first target = %q, want %q", targets[0].TmuxTarget, "revui:0.0")
	}
	if targets[0].Label != "revui:0.0  claude" {
		t.Errorf("first label = %q, want %q", targets[0].Label, "revui:0.0  claude")
	}
	if targets[1].TmuxTarget != "go:0.0" {
		t.Errorf("second target = %q, want %q", targets[1].TmuxTarget, "go:0.0")
	}
}

func TestParseTmuxPanesExcludesCurrentPid(t *testing.T) {
	input := "revui:0.0 claude 12345\n"
	// Exclude by PID matching
	targets := parseTmuxPanes(input, "%0")

	// Should still include since we filter by pane ID not PID
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
}

func TestParseTmuxPanesEmpty(t *testing.T) {
	targets := parseTmuxPanes("", "%5")

	if len(targets) != 0 {
		t.Fatalf("expected 0 targets, got %d", len(targets))
	}
}

func TestParseTmuxPanesNoClaude(t *testing.T) {
	input := "revui:0.0 nvim 12345\nrevui:1.0 lazygit 12346\n"
	targets := parseTmuxPanes(input, "%5")

	if len(targets) != 0 {
		t.Fatalf("expected 0 targets, got %d", len(targets))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/output/ -v -run TestParseTmuxPanes`
Expected: Compilation error — package and function don't exist yet.

**Step 3: Implement types and parsing**

```go
package output

import (
	"strings"
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

// parseTmuxPanes parses `tmux list-panes` output and returns Claude targets.
// Format: "session:window.pane command pid\n"
// currentPane is the value of $TMUX_PANE to exclude self.
func parseTmuxPanes(output, currentPane string) []OutputTarget {
	var targets []OutputTarget
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		paneID := fields[0]
		command := fields[1]
		if command != "claude" {
			continue
		}
		targets = append(targets, OutputTarget{
			Kind:       TargetClaude,
			Label:      paneID + "  claude",
			TmuxTarget: paneID,
		})
	}
	return targets
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/output/ -v -run TestParseTmuxPanes`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/output.go internal/output/output_test.go
git commit -m "feat: add output package with tmux pane parsing"
```

---

### Task 2: Add target detection and delivery functions

**Files:**
- Modify: `internal/output/output.go`
- Modify: `internal/output/output_test.go`

**Step 1: Write the failing test for DetectTargets**

```go
func TestDetectTargetsOutsideTmux(t *testing.T) {
	targets := DetectTargets("", "%0")

	// Should have clipboard and file, but no tmux buffer or claude
	hasClipboard := false
	hasFile := false
	for _, tgt := range targets {
		if tgt.Kind == TargetClipboard {
			hasClipboard = true
		}
		if tgt.Kind == TargetFile {
			hasFile = true
		}
		if tgt.Kind == TargetTmuxBuffer {
			t.Error("should not have tmux buffer target outside tmux")
		}
		if tgt.Kind == TargetClaude {
			t.Error("should not have claude target outside tmux")
		}
	}
	if !hasClipboard {
		t.Error("should have clipboard target")
	}
	if !hasFile {
		t.Error("should have file target")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/output/ -v -run TestDetectTargets`
Expected: Compilation error — `DetectTargets` doesn't exist.

**Step 3: Implement DetectTargets**

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DetectTargets discovers available output destinations.
// tmuxEnv is the value of $TMUX (empty if not in tmux).
// tmuxPane is the value of $TMUX_PANE.
func DetectTargets(tmuxEnv, tmuxPane string) []OutputTarget {
	var targets []OutputTarget

	if tmuxEnv != "" {
		// Detect Claude instances
		out, err := exec.Command("tmux", "list-panes", "-a", "-F",
			"#{session_name}:#{window_index}.#{pane_index} #{pane_current_command} #{pane_pid}").Output()
		if err == nil {
			targets = append(targets, parseTmuxPanes(string(out), tmuxPane)...)
		}

		targets = append(targets, OutputTarget{
			Kind:  TargetTmuxBuffer,
			Label: "tmux paste buffer",
		})
	}

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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/output/ -v -run TestDetectTargets`
Expected: PASS

**Step 5: Write the failing test for Deliver**

```go
func TestDeliverFile(t *testing.T) {
	content := "## Code Review\n\nTest content"
	target := OutputTarget{Kind: TargetFile}

	result, err := Deliver(target, content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result message")
	}
	// Result should contain the file path
	if !strings.Contains(result, "/tmp/revui-review-") {
		t.Errorf("result %q should contain temp file path", result)
	}
	// Verify file was written
	// Extract path from result message
	parts := strings.Fields(result)
	var filePath string
	for _, p := range parts {
		if strings.HasPrefix(p, "/tmp/revui-review-") {
			filePath = p
			break
		}
	}
	if filePath == "" {
		t.Fatal("could not find file path in result")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("could not read written file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
	os.Remove(filePath)
}
```

**Step 6: Implement Deliver**

```go
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
		return "", fmt.Errorf("unknown target kind: %d", target.Kind)
	}
}

func reviewFilePath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("revui-review-%d.md", time.Now().Unix()))
}

func deliverToClaude(target OutputTarget, content string) (string, error) {
	path := reviewFilePath()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write review file: %w", err)
	}
	// Send @path to the Claude pane (no Enter — user adds context before submitting)
	ref := "@" + path + " "
	cmd := exec.Command("tmux", "send-keys", "-t", target.TmuxTarget, ref)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux send-keys to %s: %w", target.TmuxTarget, err)
	}
	return fmt.Sprintf("Review sent to Claude at %s (file: %s)", target.TmuxTarget, path), nil
}

func deliverToTmuxBuffer(content string) (string, error) {
	cmd := exec.Command("tmux", "load-buffer", "-")
	cmd.Stdin = strings.NewReader(content)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux load-buffer: %w", err)
	}
	return "Review loaded into tmux paste buffer. Use prefix + ] to paste.", nil
}

func deliverToClipboard(content string) (string, error) {
	// Import clipboard at call site to avoid init() side effects in tests
	clipCmd := exec.Command("xclip", "-selection", "clipboard")
	clipCmd.Stdin = strings.NewReader(content)
	if err := clipCmd.Run(); err != nil {
		// Try xsel
		clipCmd = exec.Command("xsel", "--input", "--clipboard")
		clipCmd.Stdin = strings.NewReader(content)
		if err := clipCmd.Run(); err != nil {
			// Try wl-copy
			clipCmd = exec.Command("wl-copy")
			clipCmd.Stdin = strings.NewReader(content)
			if err := clipCmd.Run(); err != nil {
				return "", fmt.Errorf("no clipboard utility available (tried xclip, xsel, wl-copy): %w", err)
			}
		}
	}
	return "Review comments copied to clipboard.", nil
}

func deliverToFile(content string) (string, error) {
	path := reviewFilePath()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write review file: %w", err)
	}
	return fmt.Sprintf("Review written to %s", path), nil
}
```

**Step 7: Run tests**

Run: `go test ./internal/output/ -v`
Expected: PASS

**Step 8: Commit**

```bash
git add internal/output/output.go internal/output/output_test.go
git commit -m "feat: add target detection and delivery functions"
```

---

### Task 3: Create the OutputSelector TUI component

**Files:**
- Create: `internal/ui/outputselect.go`
- Create: `internal/ui/outputselect_test.go`

**Step 1: Write the failing tests**

```go
package ui

import (
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

func TestOutputSelectorNavigation(t *testing.T) {
	os := NewOutputSelector(testTargets(), 80, 24)

	// Should start at first item
	if os.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", os.cursor)
	}

	// j moves down
	os, _ = os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if os.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", os.cursor)
	}

	// k moves up
	os, _ = os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if os.cursor != 0 {
		t.Errorf("after k: cursor = %d, want 0", os.cursor)
	}

	// k at top stays at 0
	os, _ = os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if os.cursor != 0 {
		t.Errorf("k at top: cursor = %d, want 0", os.cursor)
	}
}

func TestOutputSelectorSelect(t *testing.T) {
	os := NewOutputSelector(testTargets(), 80, 24)

	// Select first item
	_, cmd := os.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should produce a command")
	}
	msg := cmd()
	selectMsg, ok := msg.(OutputSelectMsg)
	if !ok {
		t.Fatalf("expected OutputSelectMsg, got %T", msg)
	}
	if selectMsg.Target.Kind != output.TargetClaude {
		t.Errorf("selected kind = %d, want TargetClaude", selectMsg.Target.Kind)
	}
}

func TestOutputSelectorCancel(t *testing.T) {
	os := NewOutputSelector(testTargets(), 80, 24)

	_, cmd := os.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should produce a command")
	}
	msg := cmd()
	_, ok := msg.(OutputCancelMsg)
	if !ok {
		t.Fatalf("expected OutputCancelMsg, got %T", msg)
	}
}

func TestOutputSelectorViewNotEmpty(t *testing.T) {
	os := NewOutputSelector(testTargets(), 80, 24)
	view := os.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestOutputSelectorNoTargets(t *testing.T) {
	os := NewOutputSelector(nil, 80, 24)
	view := os.View()
	if view == "" {
		t.Error("expected non-empty view even with no targets")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -v -run TestOutputSelector`
Expected: Compilation error — types don't exist.

**Step 3: Implement OutputSelector**

```go
package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/output"
)

// OutputSelectMsg is sent when the user selects an output target.
type OutputSelectMsg struct {
	Target output.OutputTarget
}

// OutputCancelMsg is sent when the user cancels the output selection.
type OutputCancelMsg struct{}

// OutputSelector is a sub-model for choosing where to send the review.
type OutputSelector struct {
	targets []output.OutputTarget
	cursor  int
	width   int
	height  int
	err     string // delivery error to display
}

// NewOutputSelector creates a new output selector with the given targets.
func NewOutputSelector(targets []output.OutputTarget, width, height int) OutputSelector {
	return OutputSelector{
		targets: targets,
		width:   width,
		height:  height,
	}
}

// SetError sets an error message to display (e.g. after a failed delivery).
func (os *OutputSelector) SetError(msg string) {
	os.err = msg
}

// Update handles key messages for navigation and selection.
func (os OutputSelector) Update(msg tea.Msg) (OutputSelector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if os.cursor < len(os.targets)-1 {
				os.cursor++
			}
		case "k", "up":
			if os.cursor > 0 {
				os.cursor--
			}
		case "enter":
			if len(os.targets) > 0 && os.cursor < len(os.targets) {
				target := os.targets[os.cursor]
				return os, func() tea.Msg {
					return OutputSelectMsg{Target: target}
				}
			}
		case "q", "esc":
			return os, func() tea.Msg {
				return OutputCancelMsg{}
			}
		}
	}
	return os, nil
}

// View renders the output selector.
func (os OutputSelector) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	normalStyle := lipgloss.NewStyle()
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	b.WriteString(titleStyle.Render("Send review to:"))
	b.WriteString("\n\n")

	if len(os.targets) == 0 {
		b.WriteString("  No output targets available.\n")
	} else {
		// Render Claude targets first, then separator, then fallbacks
		hasClaude := false
		pastClaude := false
		for i, t := range os.targets {
			if t.Kind == output.TargetClaude {
				hasClaude = true
			} else if hasClaude && !pastClaude {
				pastClaude = true
				b.WriteString(separatorStyle.Render("  ── or ──"))
				b.WriteString("\n")
			}

			if i == os.cursor {
				b.WriteString(selectedStyle.Render("  > " + t.Label))
			} else {
				b.WriteString(normalStyle.Render("    " + t.Label))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	if os.err != "" {
		b.WriteString(errorStyle.Render("  Error: " + os.err))
		b.WriteString("\n\n")
	}
	b.WriteString(footerStyle.Render("  [Enter] select  [q] cancel"))
	b.WriteString("\n")

	return b.String()
}

// SetSize updates the dimensions.
func (os *OutputSelector) SetSize(width, height int) {
	os.width = width
	os.height = height
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/ -v -run TestOutputSelector`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/outputselect.go internal/ui/outputselect_test.go
git commit -m "feat: add OutputSelector TUI component"
```

---

### Task 4: Integrate OutputSelector into RootModel

**Files:**
- Modify: `internal/ui/root.go`
- Modify: `internal/ui/root_test.go`

**Step 1: Write the failing tests**

Add to `root_test.go`:

```go
func TestRootZZShowsOutputSelector(t *testing.T) {
	m := newTestRoot()

	// Add a comment so there's output
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(RootModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(RootModel)
	// Type comment and submit
	m.commentInput.input.SetValue("test comment")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(RootModel)
	// Process the CommentSubmitMsg
	updated, _ = m.Update(CommentSubmitMsg{
		FilePath: "main.go", LineNo: 1, EndLineNo: 1, Body: "test comment",
	})
	m = updated.(RootModel)

	// First Z
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m = updated.(RootModel)
	// Second Z — should show output selector, not quit
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m = updated.(RootModel)

	if m.focus != focusOutputSelect {
		t.Errorf("focus = %d, want focusOutputSelect", m.focus)
	}
	if cmd != nil {
		t.Error("should not produce a quit command")
	}
}

func TestRootZZNoCommentsQuitsDirectly(t *testing.T) {
	m := newTestRoot()

	// ZZ with no comments should quit directly
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m = updated.(RootModel)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m = updated.(RootModel)

	if !m.Finished() {
		t.Error("ZZ with no comments should finish")
	}
	if cmd == nil {
		t.Error("ZZ with no comments should produce quit command")
	}
}

func TestRootOutputSelectorCancel(t *testing.T) {
	m := newTestRoot()
	m.focus = focusOutputSelect
	m.outputSelector = NewOutputSelector(nil, 80, 24)

	updated, _ := m.Update(OutputCancelMsg{})
	m = updated.(RootModel)

	if !m.quitting {
		t.Error("cancel should set quitting")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -v -run "TestRootZZShows|TestRootZZNoComments|TestRootOutputSelector"`
Expected: Compilation errors — `focusOutputSelect`, `outputSelector` field don't exist.

**Step 3: Modify RootModel**

Changes to `internal/ui/root.go`:

1. Add `focusOutputSelect` to the `focusArea` enum:
```go
const (
	focusFileList focusArea = iota
	focusDiffViewer
	focusCommentInput
	focusOutputSelect
)
```

2. Add fields to `RootModel`:
```go
type RootModel struct {
	// ... existing fields ...
	outputSelector  OutputSelector
	deliveryResult  string // status message after delivery
}
```

3. Modify the `ZZ` handler in `handleKeyMsg` — instead of `tea.Quit`, transition to output selector:
```go
// ZZ key sequence
if key == "Z" {
	if m.pendingZ {
		m.pendingZ = false
		m.output = comment.Format(m.comments.All())
		if m.output == "" {
			// No comments — quit directly
			m.finished = true
			return m, tea.Quit
		}
		// Detect targets and show output selector
		targets := output.DetectTargets(os.Getenv("TMUX"), os.Getenv("TMUX_PANE"))
		m.outputSelector = NewOutputSelector(targets, m.width, m.height)
		m.focus = focusOutputSelect
		return m, nil
	}
	m.pendingZ = true
	return m, nil
}
```

4. Add `OutputSelectMsg` and `OutputCancelMsg` handling in `Update`:
```go
case OutputSelectMsg:
	result, err := output.Deliver(msg.Target, m.output)
	if err != nil {
		m.outputSelector.SetError(err.Error())
		return m, nil
	}
	m.deliveryResult = result
	m.finished = true
	return m, tea.Quit

case OutputCancelMsg:
	m.quitting = true
	return m, tea.Quit
```

5. Route key messages to `OutputSelector` when focused:
```go
// In the tea.KeyMsg handler, add before existing focus routing:
if m.focus == focusOutputSelect {
	var cmd tea.Cmd
	m.outputSelector, cmd = m.outputSelector.Update(msg)
	return m, cmd
}
```

6. Update `View()` to render `OutputSelector` when focused:
```go
// At the top of View(), after the error and help checks:
if m.focus == focusOutputSelect {
	return m.outputSelector.View()
}
```

7. Add `DeliveryResult()` accessor:
```go
// DeliveryResult returns the status message from the delivery (available after finish).
func (m RootModel) DeliveryResult() string {
	return m.deliveryResult
}
```

8. Handle `finishMsg` the same way (for consistency):
```go
case finishMsg:
	m.output = comment.Format(m.comments.All())
	if m.output == "" {
		m.finished = true
		return m, tea.Quit
	}
	targets := output.DetectTargets(os.Getenv("TMUX"), os.Getenv("TMUX_PANE"))
	m.outputSelector = NewOutputSelector(targets, m.width, m.height)
	m.focus = focusOutputSelect
	return m, nil
```

9. Add import for `"os"` and `"github.com/deparker/revui/internal/output"`.

**Step 4: Run tests**

Run: `go test ./internal/ui/ -v -run "TestRootZZShows|TestRootZZNoComments|TestRootOutputSelector"`
Expected: PASS

**Step 5: Run all tests to check for regressions**

Run: `go test ./internal/ui/ -v`
Expected: PASS. Note: `TestRootZZFinish` will need updating since ZZ no longer quits directly when there's output. But with `newTestRoot()` (no comments added), it should still quit directly.

**Step 6: Commit**

```bash
git add internal/ui/root.go internal/ui/root_test.go
git commit -m "feat: integrate OutputSelector into RootModel on ZZ"
```

---

### Task 5: Update main.go to use DeliveryResult

**Files:**
- Modify: `cmd/revui/main.go`

**Step 1: Update main.go**

Replace the clipboard logic with delivery result printing:

```go
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/deparker/revui/internal/git"
	"github.com/deparker/revui/internal/ui"
)

func main() {
	// ... flag parsing and git setup unchanged ...

	rm, ok := finalModel.(ui.RootModel)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unexpected model type\n")
		os.Exit(1)
	}
	if rm.Finished() && rm.DeliveryResult() != "" {
		fmt.Println(rm.DeliveryResult())
	}
}
```

This removes the `github.com/atotto/clipboard` import from `main.go` since delivery is now handled inside the TUI via `internal/output`.

**Step 2: Run go vet and build**

Run: `go vet ./... && go build ./cmd/revui`
Expected: Clean build, no errors.

**Step 3: Commit**

```bash
git add cmd/revui/main.go
git commit -m "refactor: move output delivery from main.go into TUI"
```

---

### Task 6: Update help overlay and remove unused clipboard dependency from main

**Files:**
- Modify: `internal/ui/help.go`
- Modify: `go.mod` (if `atotto/clipboard` is no longer imported anywhere)

**Step 1: Update help text**

In `help.go`, update the `ZZ` description:
```
"  ZZ          Finish review (choose output destination)\n"
```

**Step 2: Check if clipboard dependency can be removed from go.mod**

Run: `grep -r "atotto/clipboard" --include="*.go" .`

If only `internal/output/output.go` uses clipboard utilities directly via `os/exec` (not the `atotto/clipboard` package), then:

Run: `go mod tidy`

This will remove the unused `atotto/clipboard` dependency.

**Step 3: Run all tests**

Run: `go test ./... -v`
Expected: All pass.

**Step 4: Commit**

```bash
git add internal/ui/help.go go.mod go.sum
git commit -m "chore: update help text and remove unused clipboard dependency"
```

---

### Task 7: Manual smoke test

**Steps:**

1. Build: `go build ./cmd/revui`
2. Run in a tmux session with Claude Code running in another pane: `./revui`
3. Add a comment on any line (`c`, type text, Enter)
4. Press `ZZ` — verify the output selector appears
5. Verify Claude instances are listed
6. Select a Claude instance — verify `@/tmp/revui-review-*.md` appears in the Claude pane
7. Test fallbacks: select "System clipboard", "tmux paste buffer", "Write to file"
8. Test cancel: press `q` on the selector
9. Test with no comments: `ZZ` should quit directly
10. Test outside tmux: run without tmux, verify only clipboard and file options appear
