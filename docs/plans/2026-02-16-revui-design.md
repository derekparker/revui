# revui — TUI Code Review Tool

## Overview

revui is a terminal-based code review tool built in Go with Bubble Tea. It
displays the diff between a feature branch and a base branch, lets the user
add inline comments on specific lines, and copies the formatted comments to the
system clipboard for pasting into a coding agent (e.g., Claude Code).

## Goals

- Fast, keyboard-driven diff review workflow with vim-style navigation
- Syntax-highlighted diffs (unified and side-by-side, toggleable)
- Inline commenting with line/range targeting
- Structured clipboard output for AI agent consumption

## Non-Goals

- GitHub/GitLab integration (local git only)
- Persistent comment storage (in-memory only, output on finish)
- Multi-repo support

## Architecture

### Approach: Composable Bubble Tea Sub-Models

Each UI panel is its own Bubble Tea model with `Init`, `Update`, and `View`.
A root model orchestrates focus, routes messages, and holds shared state.

```
┌─────────────────────────────────────────────────────┐
│                    Root Model                        │
│  (orchestrates focus, routes messages, manages git)  │
├──────────────┬──────────────────────────────────────┤
│  File List   │           Diff Viewer                 │
│  (sub-model) │  (sub-model with viewport)            │
│              │                                       │
│  ▸ main.go   │  @@ -10,6 +10,8 @@                   │
│    config.go  │   func foo() {                        │
│    util.go    │  -    old code                        │
│              │  +    new code            [comment]   │
│              │       unchanged                       │
├──────────────┴──────────────────────────────────────┤
│  Status Bar / Comment Input                          │
│  [c]omment  [v]isual  [Tab]view  [ZZ]done  [?]help  │
└─────────────────────────────────────────────────────┘
```

### Components

| Component      | Type           | Responsibility                                      |
|----------------|----------------|-----------------------------------------------------|
| `RootModel`    | Bubble Tea     | Focus management, key routing, git state, comments  |
| `FileList`     | Sub-model      | Changed file list, selection, filtering             |
| `DiffViewer`   | Sub-model      | Diff rendering, scrolling, line selection            |
| `CommentInput` | Sub-model      | Text input overlay for adding/editing comments       |
| `StatusBar`    | Render-only    | Keybindings display and current state               |

### Package Layout

```
cmd/revui/          main entry point, CLI flag parsing
internal/
  git/              wraps git commands, parses diffs
  ui/               all Bubble Tea models and rendering
  comment/          comment data model and clipboard formatting
  syntax/           chroma-based syntax highlighting
```

## Data Flow

### Startup

1. User runs `revui` or `revui --base main` from within a git repo
2. Detect current branch and base branch (default: `main`, override with `--base`)
3. Run `git diff --name-status <base>..HEAD` to get changed file list
4. Display file list; selecting a file loads its diff lazily

### Git Integration (`internal/git`)

- Shells out to `git` via `os/exec` (no git library dependency)
- Parses unified diff output into structured models

### Data Model

```go
type FileDiff struct {
    Path   string
    Status string // A, M, D, R
    Hunks  []Hunk
}

type Hunk struct {
    Header string
    Lines  []Line
}

type Line struct {
    Content   string
    Type      LineType // Added, Removed, Context
    OldLineNo int
    NewLineNo int
}

type Comment struct {
    FilePath  string
    StartLine int
    EndLine   int
    LineType  LineType
    Body      string
}
```

Diffs are loaded per-file on selection, not all at once.

## Keybindings (Vim-style)

### Navigation

| Key              | Action                                    |
|------------------|-------------------------------------------|
| `j` / `k`        | Move down/up in file list or diff lines   |
| `h` / `l`        | Switch focus between file list and diff   |
| `gg`             | Jump to top                               |
| `G`              | Jump to bottom                            |
| `Ctrl+d`/`Ctrl+u`| Half-page down/up                         |
| `Ctrl+f`/`Ctrl+b`| Full page down/up                         |
| `{` / `}`        | Jump to previous/next hunk                |
| `/`              | Search within diff                        |
| `n` / `N`        | Next/previous search match                |

### Review Actions

| Key         | Action                                       |
|-------------|----------------------------------------------|
| `c`         | Add/edit comment on current line              |
| `v`         | Visual mode to select line range, then `c`    |
| `Esc`       | Cancel current action / exit visual mode      |
| `dd`        | Delete comment on current line                |
| `]c` / `[c` | Jump to next/previous comment                |

### View & Global

| Key     | Action                                            |
|---------|---------------------------------------------------|
| `Tab`   | Toggle unified / side-by-side diff view           |
| `Enter` | Open selected file's diff (in file list)          |
| `q`     | Quit without copying                              |
| `ZZ`    | Finish review — copy comments to clipboard, quit  |
| `?`     | Show help overlay                                 |

## Syntax Highlighting

- Uses `alecthomas/chroma` for language-aware syntax highlighting
- File extension determines the lexer
- Terminal-friendly dark theme (monokai or dracula)
- Applied lazily per-viewport for performance with large files
- Added lines: green-tinted background over syntax colors
- Removed lines: red-tinted background over syntax colors
- Line numbers in a gutter column
- Comment indicators in the gutter next to commented lines
- Hunk headers styled distinctly (dimmed)

### Side-by-Side Mode

- Two columns via Lipgloss: left = old lines, right = new lines
- Blank lines fill gaps for additions/deletions
- Line numbers correspond to old/new file respectively

## Comment Workflow

### Adding a Comment

1. Scroll to a line in the diff viewer
2. Press `c` to comment on current line (or `v` to select range, then `c`)
3. Text input overlay appears at the bottom
4. Type comment, `Enter` to save, `Esc` to cancel
5. Commented lines show a visual indicator in the gutter

### Managing Comments

- `c` on a commented line edits the existing comment
- `dd` on a commented line deletes it (with confirmation)
- `]c` / `[c` to navigate between comments

### Clipboard Output (on `ZZ`)

All comments are formatted as structured markdown:

```markdown
## Code Review Comments

### path/to/file.go

**Lines 42-45 (added):**
\```go
func newHelper() {
    // ...
}
\```
**Comment:** This function doesn't handle the error case. Please add error
handling that returns the error to the caller.

---

### path/to/other.go

**Line 18 (modified):**
\```go
timeout := 30 * time.Second
\```
**Comment:** This timeout seems too high for a health check. Consider 5s.
```

Output includes: file path, line numbers, change type, code snippet context,
and the comment text.

### Future Enhancement

- `--output <file>` flag to write comments to a file instead of clipboard
- Stdout output when piped (detect non-TTY)

## Error Handling

- **Not in a git repo:** exit with clear error message
- **Base branch doesn't exist:** error with suggestion to use `--base`
- **Binary files:** show "Binary file changed" placeholder
- **Very large diffs:** lazy highlighting per-viewport
- **No changes:** message that there are no differences
- **Clipboard failure:** fall back to printing to stdout with a warning

## Testing Strategy

- **`internal/git`:** unit tests with fixture diff strings (no real git repo)
- **`internal/comment`:** unit tests for CRUD and clipboard format output
- **`internal/syntax`:** unit tests for chroma integration with known inputs
- **`internal/ui`:** test each sub-model's Update with simulated key messages,
  verify state transitions and View output
- **Integration:** full app model with mock git backend, end-to-end flow

## Dependencies

- `charmbracelet/bubbletea` — TUI framework
- `charmbracelet/lipgloss` — terminal styling and layout
- `charmbracelet/bubbles` — viewport, list, textinput components
- `alecthomas/chroma` — syntax highlighting
- `atotto/clipboard` — cross-platform clipboard access
