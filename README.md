# revui

A terminal-based code review tool for git diffs. Navigate changes with vim-style keybindings, add inline comments, and copy the formatted review to your clipboard — ready to paste into an AI coding assistant or anywhere else.

```
 revui — main → feature/auth
┌──────────────────────────┬───────────────────────────────────────────┐
│ M  internal/auth/auth.go │  @@ -12,6 +12,15 @@                     │
│ A  internal/auth/token.go│    func Authenticate(user string) error {│
│ M  cmd/server/main.go    │  +     token, err := generateToken(user) │
│                          │  +     if err != nil {                   │
│                          │  +         return fmt.Errorf(...)        │
│                          │  +     }                                 │
│                          │  ● This should validate the user first   │
│                          │                                          │
└──────────────────────────┴───────────────────────────────────────────┘
 [c]omment  [v]isual  [Tab]view  [q]uit  [ZZ]done  [?]help  │  1 comments
```

## Install

```bash
go install github.com/deparker/revui/cmd/revui@latest
```

Or build from source:

```bash
git clone https://github.com/deparker/revui.git
cd revui
go build ./cmd/revui
```

## Usage

Run from any git repository with uncommitted or branched changes:

```bash
revui                      # auto-detect base branch from origin/HEAD
revui --base main          # diff against a specific branch
revui --remote upstream    # auto-detect base from a different remote
```

When you finish reviewing (`ZZ`), your comments are formatted as markdown and copied to the clipboard:

```markdown
## Code Review Comments

### internal/auth/auth.go

**Line 15 (added):**
```go
token, err := generateToken(user)
```
**Comment:** This should validate the user first

---

### cmd/server/main.go

**Lines 42-45:**
```go
if err != nil {
    log.Fatal(err)
}
```
**Comment:** Use log.Error and return instead of Fatal in a handler
```

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `h` / `l` | Switch to file list / diff panel |
| `Enter` | Open selected file's diff |
| `G` / `gg` | Jump to bottom / top |
| `Ctrl+d` / `Ctrl+u` | Half-page down / up |
| `Ctrl+f` / `Ctrl+b` | Full-page down / up |
| `[` / `]` | Jump to prev / next change |
| `{` / `}` | Jump to prev / next hunk |

### Commenting

| Key | Action |
|-----|--------|
| `c` | Add or edit comment on current line |
| `v` | Enter visual mode (select a range of lines) |
| `v` then `c` | Comment on selected range |
| `D` | Delete comment on current line |
| `]c` / `[c` | Jump to next / prev comment |

### Views and Actions

| Key | Action |
|-----|--------|
| `Tab` | Toggle unified / side-by-side view |
| `/` | Search in diff |
| `n` / `N` | Next / prev search result |
| `ZZ` | Finish review and copy comments to clipboard |
| `q` | Quit without copying |
| `?` | Toggle help overlay |

## Requirements

- Go 1.25+
- `git` in your PATH
- A terminal with 256-color support
