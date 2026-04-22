# File List Toggle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `ctrl+h` keybinding to `RootModel` that toggles the file list panel, giving the diff viewer full terminal width when the panel is hidden.

**Architecture:** Add `hideFileList bool` to `RootModel` and a `diffViewerWidth()` helper that returns either the full width or the current split width. The `ctrl+h` handler flips the flag, resizes the diff viewer immediately, and shifts focus away from the file list if it is the active panel. `View()` skips rendering the file list panel when the flag is set.

**Tech Stack:** Go, Bubble Tea (`github.com/charmbracelet/bubbletea`), lipgloss (`github.com/charmbracelet/lipgloss`)

---

## Files

- Modify: `internal/ui/root.go` — add field, helper, key handler, view logic
- Modify: `internal/ui/help.go` — add `Ctrl+h` entry under "Views"
- Modify: `internal/ui/root_test.go` — new tests for toggle behavior

---

### Task 1: Write failing tests for the toggle feature

**Files:**
- Modify: `internal/ui/root_test.go`

- [ ] **Step 1: Add the failing tests at the bottom of `root_test.go`**

Add these tests after the last existing test in `internal/ui/root_test.go`:

```go
func TestFileListToggle_InitiallyVisible(t *testing.T) {
	m := newTestRoot()
	if m.hideFileList {
		t.Error("file list should be visible by default")
	}
}

func TestFileListToggle_CtrlHHides(t *testing.T) {
	m := newTestRoot()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	m = updated.(RootModel)
	if !m.hideFileList {
		t.Error("ctrl+h should set hideFileList = true")
	}
}

func TestFileListToggle_CtrlHToggles(t *testing.T) {
	m := newTestRoot()
	// Hide
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	m = updated.(RootModel)
	// Show again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	m = updated.(RootModel)
	if m.hideFileList {
		t.Error("second ctrl+h should restore hideFileList = false")
	}
}

func TestFileListToggle_FocusShiftsWhenFileListFocused(t *testing.T) {
	m := newTestRoot()
	// Initial focus is on file list
	if m.focus != focusFileList {
		t.Fatal("expected initial focus on file list")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	m = updated.(RootModel)
	if m.focus != focusDiffViewer {
		t.Error("hiding file list while focused should shift focus to diff viewer")
	}
}

func TestFileListToggle_FocusUnchangedWhenDiffFocused(t *testing.T) {
	m := newTestRoot()
	// Move focus to diff viewer first
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(RootModel)
	if m.focus != focusDiffViewer {
		t.Fatal("expected focus on diff viewer")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	m = updated.(RootModel)
	if m.focus != focusDiffViewer {
		t.Error("focus should remain on diff viewer when hiding file list from diff viewer")
	}
}

func TestFileListToggle_HKeyNoOpWhenHidden(t *testing.T) {
	m := newTestRoot()
	// Switch to diff viewer, then hide file list
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(RootModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	m = updated.(RootModel)
	if m.focus != focusDiffViewer {
		t.Fatal("expected focus on diff viewer")
	}
	// h should not shift focus to hidden file list
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(RootModel)
	if m.focus != focusDiffViewer {
		t.Error("h should be no-op when file list is hidden")
	}
}

func TestFileListToggle_DiffViewerWidthHelper(t *testing.T) {
	m := newTestRoot()
	// m.width = 80, m.fileListWidth = 30, border = 3
	wantVisible := 80 - 30 - 3
	wantHidden := 80

	if got := m.diffViewerWidth(); got != wantVisible {
		t.Errorf("diffViewerWidth() visible = %d, want %d", got, wantVisible)
	}

	m.hideFileList = true
	if got := m.diffViewerWidth(); got != wantHidden {
		t.Errorf("diffViewerWidth() hidden = %d, want %d", got, wantHidden)
	}
}

func TestFileListToggle_ViewOmitsFileListWhenHidden(t *testing.T) {
	m := newTestRoot()

	// When visible, the file list cursor arrow is present
	view := m.View()
	if !strings.Contains(view, "▸") {
		t.Error("visible file list should contain cursor arrow ▸")
	}

	// When hidden, the file list is not rendered
	m.hideFileList = true
	view = m.View()
	if strings.Contains(view, "▸") {
		t.Error("hidden file list should not render cursor arrow ▸")
	}
}
```

- [ ] **Step 2: Run the tests to confirm they fail**

```bash
cd /home/deparker/Code/revui && go test ./internal/ui/ -run TestFileListToggle -v
```

Expected: compile error (`m.hideFileList undefined`, `m.diffViewerWidth undefined`)

---

### Task 2: Add `hideFileList` field, `diffViewerWidth()` helper, and update `WindowSizeMsg` handling

**Files:**
- Modify: `internal/ui/root.go`

- [ ] **Step 1: Add `hideFileList bool` to the `RootModel` struct**

In `internal/ui/root.go`, in the `RootModel` struct (around line 59), add the field after `fileListWidth`:

```go
	fileListWidth     int
	hideFileList      bool
```

- [ ] **Step 2: Add the `diffViewerWidth()` helper method**

Add this method after the `loadFileDiff` method (around line 509):

```go
// diffViewerWidth returns the width for the diff viewer panel.
// When the file list is hidden it gets the full terminal width.
func (m RootModel) diffViewerWidth() int {
	if m.hideFileList {
		return m.width
	}
	return m.width - m.fileListWidth - 3
}
```

- [ ] **Step 3: Replace the inline diff-width expression in `WindowSizeMsg` handling**

In `Update()`, the `tea.WindowSizeMsg` case currently reads (around line 182):

```go
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.fileList.SetSize(m.fileListWidth, m.height-2)
		m.diffViewer.SetSize(m.width-m.fileListWidth-3, m.height-2)
		m.commentInput.SetWidth(m.width)
		return m, nil
```

Replace `m.width-m.fileListWidth-3` with `m.diffViewerWidth()`:

```go
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.fileList.SetSize(m.fileListWidth, m.height-2)
		m.diffViewer.SetSize(m.diffViewerWidth(), m.height-2)
		m.commentInput.SetWidth(m.width)
		return m, nil
```

- [ ] **Step 4: Run the relevant tests to check progress**

```bash
cd /home/deparker/Code/revui && go test ./internal/ui/ -run TestFileListToggle -v
```

Expected: `TestFileListToggle_DiffViewerWidthHelper` passes; others still fail (`ctrl+h` not handled yet).

---

### Task 3: Add `ctrl+h` key handler and guard the `h` key

**Files:**
- Modify: `internal/ui/root.go`

- [ ] **Step 1: Add `ctrl+h` case to `handleKeyMsg`**

In `handleKeyMsg`, the `switch key {` block starts around line 362. Add a new case before the existing `"q"` case:

```go
	case "ctrl+h":
		m.hideFileList = !m.hideFileList
		if m.hideFileList && m.focus == focusFileList {
			m.focus = focusDiffViewer
		}
		m.diffViewer.SetSize(m.diffViewerWidth(), m.height-2)
		return m, nil
```

- [ ] **Step 2: Guard the existing `h` key case**

The existing `"h"` case currently reads (around line 392):

```go
	case "h":
		if m.focus == focusDiffViewer {
			m.focus = focusFileList
		}
		return m, nil
```

Add the `!m.hideFileList` guard:

```go
	case "h":
		if m.focus == focusDiffViewer && !m.hideFileList {
			m.focus = focusFileList
		}
		return m, nil
```

- [ ] **Step 3: Run the toggle tests**

```bash
cd /home/deparker/Code/revui && go test ./internal/ui/ -run TestFileListToggle -v
```

Expected: all tests pass except `TestFileListToggle_ViewOmitsFileListWhenHidden` (View not updated yet).

---

### Task 4: Update `View()` to conditionally render the file list panel

**Files:**
- Modify: `internal/ui/root.go`

- [ ] **Step 1: Replace the panel rendering block in `View()`**

In `View()`, the section that builds and joins the two panels currently reads (around line 592):

```go
	// File list panel
	fileListPanel := lipgloss.NewStyle().
		Width(m.fileListWidth).
		Height(m.height - 3).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Render(m.fileList.View())

	// Diff viewer panel
	diffPanel := lipgloss.NewStyle().
		Width(m.width - m.fileListWidth - 3).
		Height(m.height - 3).
		Render(m.diffViewer.View())

	// Main content
	content := lipgloss.JoinHorizontal(lipgloss.Top, fileListPanel, diffPanel)
	b.WriteString(content)
```

Replace it with:

```go
	// Diff viewer panel — expands to full width when file list is hidden
	diffPanel := lipgloss.NewStyle().
		Width(m.diffViewerWidth()).
		Height(m.height - 3).
		Render(m.diffViewer.View())

	var content string
	if m.hideFileList {
		content = diffPanel
	} else {
		fileListPanel := lipgloss.NewStyle().
			Width(m.fileListWidth).
			Height(m.height - 3).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Render(m.fileList.View())
		content = lipgloss.JoinHorizontal(lipgloss.Top, fileListPanel, diffPanel)
	}
	b.WriteString(content)
```

- [ ] **Step 2: Run all toggle tests**

```bash
cd /home/deparker/Code/revui && go test ./internal/ui/ -run TestFileListToggle -v
```

Expected: all 8 `TestFileListToggle_*` tests pass.

- [ ] **Step 3: Run the full test suite**

```bash
cd /home/deparker/Code/revui && go test ./... -v 2>&1 | tail -20
```

Expected: all tests pass, no compile errors.

---

### Task 5: Update the help overlay and commit

**Files:**
- Modify: `internal/ui/help.go`

- [ ] **Step 1: Add `Ctrl+h` entry under the "Views" section**

In `help.go`, the Views section currently reads (around line 33):

```go
		"Views\n" +
		"  Tab         Toggle unified/side-by-side view\n" +
		"  /           Search in diff\n" +
		"  n/N         Next/prev search result\n" +
```

Add the `Ctrl+h` line:

```go
		"Views\n" +
		"  Tab         Toggle unified/side-by-side view\n" +
		"  Ctrl+h      Toggle file list\n" +
		"  /           Search in diff\n" +
		"  n/N         Next/prev search result\n" +
```

- [ ] **Step 2: Run the full test suite one final time**

```bash
cd /home/deparker/Code/revui && go test ./...
```

Expected: `ok` for all packages, no failures.

- [ ] **Step 3: Vet and format**

```bash
cd /home/deparker/Code/revui && go vet ./... && go fmt ./...
```

Expected: no output (no issues).

- [ ] **Step 4: Commit**

```bash
cd /home/deparker/Code/revui && git add internal/ui/root.go internal/ui/help.go internal/ui/root_test.go && git commit -m "feat: add ctrl+h toggle for file list panel"
```
