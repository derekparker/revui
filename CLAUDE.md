# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

revui is a terminal-based code review tool for git diffs, built with Go and the Bubble Tea TUI framework. It provides vim-style navigation, side-by-side and unified diff views, inline commenting, and copies formatted markdown to clipboard on finish — designed for pasting review comments into AI coding assistants.

## Commands

```bash
# Build
go build ./cmd/revui

# Run
./revui                    # auto-detect base branch from origin/HEAD
./revui --base main        # explicit base branch
./revui --remote upstream  # auto-detect from a different remote

# Test
go test ./...                          # all tests
go test ./internal/ui/ -v              # single package
go test -run TestDiffView ./internal/ui/  # single test

# Benchmarks
go test ./... -bench=. -benchmem -count=3

# Lint/format
go vet ./...
go fmt ./...
```

## Architecture

The app follows Bubble Tea's composable model pattern. Each UI component implements `Init()`, `Update()`, `View()`.

**Package structure:**

- `cmd/revui/` — Entry point. Parses flags, validates git repo, auto-detects base branch, runs the TUI, copies comments to clipboard on finish (`ZZ`).
- `internal/git/` — Git operations via `os/exec`. `Runner` shells out to git; `parse.go` parses unified diff output into structured types (`FileDiff` → `Hunk` → `Line`). The `GitRunner` interface (defined in `internal/ui/root.go`) enables mock-based testing.
- `internal/comment/` — In-memory `Store` for review comments with O(1) lookup by file+line via map index. `format.go` renders comments as markdown.
- `internal/ui/` — All TUI components:
  - `root.go` — `RootModel` orchestrates focus routing between `FileList`, `DiffViewer`, and `CommentInput`. Handles global keys (Tab for view toggle, `ZZ` to finish, `q` to quit).
  - `filelist.go` — Left panel file list with j/k navigation.
  - `diffview.go` — Main diff rendering (unified and side-by-side). Hot path (~710 lines). Flattens hunks into `[]diffLine` for scrolling. Handles visual mode line selection, search (`/`, `n`, `N`), and comment navigation (`]c`, `[c`).
  - `sidebyside.go` — Pairs left/right lines for side-by-side rendering.
  - `commentinput.go` — Modal text input overlay.
  - `help.go` — Help overlay (`?`).

**Data flow:** `main.go` → `RootModel` → routes keys to focused sub-model → `DiffViewer` calls `GitRunner` to lazy-load diffs per file → comments stored in `comment.Store` → on `ZZ`, `comment.Format()` produces markdown → copied to clipboard.

## Key Conventions

- **Dependency injection:** `GitRunner` interface lets UI tests use `mockGitRunner` instead of real git.
- **Lazy loading:** Diffs are fetched per-file on selection, not upfront.
- **Performance-conscious rendering:** `strings.Builder` with `Grow()`, fixed-size byte arrays for line number formatting, reused empty lipgloss styles. Changes to `diffview.go` rendering should be benchmarked.
- **Test helpers:** `setupTestRepo()` creates real git repos in temp dirs for git package tests. `makeTestDiff()` and `newTestRoot()` for UI tests. Tests are table-driven.

## Keybindings Reference

`j/k` navigate, `Tab` toggles unified/side-by-side, `v` enters visual mode, `c` adds comment, `]c/[c` next/prev comment, `/` search, `ZZ` finish (copy to clipboard), `q` quit, `?` help.
