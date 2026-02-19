# Uncommitted Changes Mode

## Goal

When the working tree has uncommitted changes (staged, unstaged, or untracked files), show those changes for review instead of the branch-vs-base diff. When the repo is clean, fall back to the current branch comparison behavior.

## Requirements

- Auto-detect uncommitted changes on startup (no new CLI flags)
- Show `git diff HEAD` for tracked files (staged + unstaged combined)
- Show untracked files as all-added diffs (full file content)
- Skip binary files in diff view but show them in file list with placeholder, still commentable
- Header reads "revui — uncommitted changes" in this mode
- Clean repo falls through to existing branch-comparison behavior unchanged

## Approach

Dual-mode with new `GitRunner` methods and a mode flag on `RootModel`.

## Design

### 1. Detection & Mode Selection (`main.go`)

Run `git status --porcelain` to check for uncommitted/untracked changes:
- Non-empty output: uncommitted mode. Skip base branch detection/validation.
- Empty output: branch mode (current behavior, unchanged).

The `--base` and `--remote` flags remain but are ignored when uncommitted changes are detected.

### 2. Git Package Changes (`internal/git/`)

Three new methods on `Runner`:

**`HasUncommittedChanges() bool`** — Runs `git status --porcelain`, returns true if non-empty.

**`UncommittedFiles() ([]ChangedFile, error)`** — Combines:
- `git diff HEAD --name-status` for tracked files with changes
- `git ls-files --others --exclude-standard` for untracked files (status "A")
- `git diff HEAD --numstat` for binary detection (binary files show `-\t-\tpath`; marked with status "B")
- Deduplicates across sources

**`UncommittedFileDiff(path string) (*FileDiff, error)`** — Two paths:
- Tracked files: `git diff HEAD -- <path>`, parsed with existing `ParseDiff`
- Untracked files: read file content, construct synthetic `FileDiff` with single hunk, all lines `LineAdded`, line numbers starting at 1
- Binary files: return `FileDiff` with empty `Hunks` and `Status: "B"`

Binary detection for untracked files uses null-byte check in first 8KB (same heuristic as git).

### 3. UI Changes (`internal/ui/`)

**Mode field on `RootModel`:**
```go
type reviewMode int
const (
    modeBranch      reviewMode = iota
    modeUncommitted
)
```

**`NewRootModel` changes:** Accepts mode parameter. In uncommitted mode:
- Calls `UncommittedFiles()` instead of `ChangedFiles(base)`
- Calls `UncommittedFileDiff(path)` instead of `FileDiff(base, path)`

**`loadFileDiff(path string)` helper:** Encapsulates the mode switch for all 4 diff-loading call sites in `root.go`.

**Header:** Branch mode: `"revui — main -> feature-branch"`. Uncommitted mode: `"revui — uncommitted changes"`.

**Binary placeholder:** When `DiffViewer` receives a `FileDiff` with no hunks and status "B", renders "Binary file — cannot display diff". Commenting still works (file path + line 0).

**File list:** `FileStatusString` gains case for "B" -> "binary".

### 4. Interface Changes

`GitRunner` interface gains three new methods:
```go
HasUncommittedChanges() bool
UncommittedFiles() ([]ChangedFile, error)
UncommittedFileDiff(path string) (*FileDiff, error)
```

Existing methods unchanged. Mock in tests adds stubs.

### 5. Testing

**Git package:** Real temp repos via `setupTestRepo()`. Test staged, unstaged, untracked, and binary files for both `UncommittedFiles` and `UncommittedFileDiff`.

**UI package:** Extend `mockGitRunner` with new methods. Test uncommitted mode header, file list, diff loading, binary placeholder, and commenting on binary files.

No changes to existing tests.
