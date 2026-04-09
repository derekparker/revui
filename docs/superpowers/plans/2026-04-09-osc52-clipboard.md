# OSC 52 Clipboard Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace xclip/wl-copy clipboard utilities with OSC 52 escape sequences for universal clipboard support across local and remote terminals.

**Architecture:** Modify `deliverToClipboard()` to write OSC 52 escape sequences to stderr instead of shelling out to external utilities. Uses existing `github.com/aymanbagabas/go-osc52/v2` dependency from Bubble Tea.

**Tech Stack:** Go 1.25, go-osc52/v2 library

---

## File Structure

**Modified files:**
- `internal/output/output.go` - Replace `deliverToClipboard()` implementation (lines 155-185)
- `internal/output/output_test.go` - Add tests for OSC 52 clipboard delivery
- `internal/ui/help.go` - Update help text to mention OSC 52 clipboard support

**No new files created.**

---

### Task 1: Replace deliverToClipboard() with OSC 52

**Files:**
- Modify: `internal/output/output.go:155-185`
- Test: `internal/output/output_test.go`

- [ ] **Step 1: Write the failing test**

Add this test to `internal/output/output_test.go` after the `TestDeliverFile` function:

```go
func TestDeliverClipboard(t *testing.T) {
	content := "# Code Review\n\nTest content for clipboard"
	target := OutputTarget{
		Kind:  TargetClipboard,
		Label: "System clipboard",
	}

	msg, err := Deliver(target, content)
	if err != nil {
		t.Fatalf("Deliver failed: %v", err)
	}

	// Check that message indicates OSC 52 was used
	if !strings.Contains(msg, "OSC 52") {
		t.Errorf("message %q does not mention OSC 52", msg)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/output/ -run TestDeliverClipboard -v`

Expected: FAIL - message does not contain "OSC 52" (current implementation returns "Review comments copied to clipboard.")

- [ ] **Step 3: Add OSC 52 import**

Add import to `internal/output/output.go` at the top of the file:

```go
import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aymanbagabas/go-osc52/v2"
)
```

- [ ] **Step 4: Replace deliverToClipboard() implementation**

Replace the entire `deliverToClipboard()` function in `internal/output/output.go` (lines 155-185) with:

```go
// deliverToClipboard copies content to clipboard using OSC 52 escape sequences.
func deliverToClipboard(content string) (string, error) {
	seq := osc52.New(content)
	_, err := fmt.Fprint(os.Stderr, seq)
	if err != nil {
		return "", fmt.Errorf("failed to write OSC 52 sequence: %w", err)
	}
	return "Review copied to clipboard via OSC 52.", nil
}
```

- [ ] **Step 5: Remove unused bytes import**

Since we removed the `bytes.Buffer` usage, remove the `bytes` import from the top of `internal/output/output.go`:

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aymanbagabas/go-osc52/v2"
)
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/output/ -run TestDeliverClipboard -v`

Expected: PASS

- [ ] **Step 7: Run all output tests**

Run: `go test ./internal/output/ -v`

Expected: All tests PASS

- [ ] **Step 8: Commit**

```bash
git add internal/output/output.go internal/output/output_test.go
git commit -m "feat: replace clipboard utilities with OSC 52 escape sequences

Replace xclip/wl-copy/xsel utility chain with OSC 52 escape sequences
for universal clipboard support across local and remote terminals.

Uses existing go-osc52/v2 library from Bubble Tea dependency tree.
Works over SSH, in tmux/zellij, and any OSC 52-capable terminal.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Add edge case tests

**Files:**
- Test: `internal/output/output_test.go`

- [ ] **Step 1: Write test for empty content**

Add this test to `internal/output/output_test.go` after `TestDeliverClipboard`:

```go
func TestDeliverClipboardEmpty(t *testing.T) {
	content := ""
	target := OutputTarget{
		Kind:  TargetClipboard,
		Label: "System clipboard",
	}

	msg, err := Deliver(target, content)
	if err != nil {
		t.Fatalf("Deliver with empty content failed: %v", err)
	}

	if !strings.Contains(msg, "OSC 52") {
		t.Errorf("message %q does not mention OSC 52", msg)
	}
}
```

- [ ] **Step 2: Write test for large content**

Add this test to `internal/output/output_test.go` after `TestDeliverClipboardEmpty`:

```go
func TestDeliverClipboardLarge(t *testing.T) {
	// Create 50KB of content (well within typical OSC 52 limits)
	content := strings.Repeat("# Code Review Comment\n", 2000)
	target := OutputTarget{
		Kind:  TargetClipboard,
		Label: "System clipboard",
	}

	msg, err := Deliver(target, content)
	if err != nil {
		t.Fatalf("Deliver with large content failed: %v", err)
	}

	if !strings.Contains(msg, "OSC 52") {
		t.Errorf("message %q does not mention OSC 52", msg)
	}
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./internal/output/ -run "TestDeliverClipboard" -v`

Expected: All 3 clipboard tests PASS (TestDeliverClipboard, TestDeliverClipboardEmpty, TestDeliverClipboardLarge)

- [ ] **Step 4: Run all tests**

Run: `go test ./... -v`

Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/output/output_test.go
git commit -m "test: add edge case tests for OSC 52 clipboard

Add tests for empty content and large content (50KB) to ensure
OSC 52 clipboard delivery handles edge cases gracefully.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Update help text for OSC 52 clipboard

**Files:**
- Modify: `internal/ui/help.go:36-38`

- [ ] **Step 1: Update help text**

Modify the "Actions" section in `internal/ui/help.go` (around line 36-38) to add clipboard information:

```go
	"Actions\n" +
	"  ZZ          Finish review (choose output destination)\n" +
	"              • Clipboard uses OSC 52 (works over SSH)\n" +
	"  q           Quit without copying\n" +
	"  ?           Toggle this help\n" +
```

- [ ] **Step 2: Build to verify no compilation errors**

Run: `go build ./cmd/revui`

Expected: Successful build, no errors

- [ ] **Step 3: Commit**

```bash
git add internal/ui/help.go
git commit -m "docs: update help text to mention OSC 52 clipboard

Add note in help overlay that clipboard option uses OSC 52,
which works over SSH connections and remote terminals.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Final verification

**Files:**
- All modified files

- [ ] **Step 1: Run all tests**

Run: `go test ./... -v`

Expected: All tests PASS

- [ ] **Step 2: Build the application**

Run: `go build ./cmd/revui`

Expected: Successful build

- [ ] **Step 3: Verify git status**

Run: `git status`

Expected: Clean working tree (all changes committed)

- [ ] **Step 4: Review commit history**

Run: `git log --oneline -4`

Expected: See 3 new commits:
1. docs: update help text to mention OSC 52 clipboard
2. test: add edge case tests for OSC 52 clipboard
3. feat: replace clipboard utilities with OSC 52 escape sequences

---

## Manual Testing Checklist

After implementation, manually test in these environments:

- [ ] Local terminal (verify clipboard works)
- [ ] SSH session (verify OSC 52 passes through)
- [ ] Inside tmux (verify tmux wrapping works)
- [ ] Inside zellij (verify zellij passthrough works)
- [ ] Terminal without OSC 52 support (verify graceful degradation - no error, just silent fail)

## Success Criteria

- [ ] All automated tests pass
- [ ] Code compiles successfully
- [ ] Clipboard copying works in OSC 52-capable terminals
- [ ] No external clipboard utility dependencies
- [ ] Code is simpler than previous implementation
- [ ] Help text documents clipboard mechanism
