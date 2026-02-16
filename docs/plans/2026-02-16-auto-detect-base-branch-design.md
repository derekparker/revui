# Auto-Detect Default Base Branch

**Goal:** Instead of hardcoding `--base` to `"main"`, auto-detect the remote's default branch using `git symbolic-ref`.

**Approach:** Use `git symbolic-ref refs/remotes/<remote>/HEAD` to detect the default branch for the configured remote. Fall back to `"main"` if detection fails. Add a `--remote` flag (default: `"origin"`) for configurability.

## Behavior

1. If `--base` is explicitly provided, use it directly (no detection).
2. If `--base` is not provided:
   - Run `git symbolic-ref refs/remotes/<remote>/HEAD`
   - Parse `refs/remotes/<remote>/<branch>` to extract `<branch>`
   - If that fails (ref doesn't exist, remote not configured), fall back to `"main"`

## Flag Combinations

```
revui                                    # auto-detect base from origin
revui --base develop                     # explicit base, no detection
revui --remote upstream                  # auto-detect base from upstream
revui --remote upstream --base develop   # explicit base, remote ignored
```

## Changes

- `internal/git/git.go` — add `DefaultBranch(remote string) string` method
- `internal/git/git_test.go` — add test for `DefaultBranch`
- `cmd/revui/main.go` — add `--remote` flag, call `DefaultBranch` when `--base` not explicitly set

## Trade-offs

- `symbolic-ref` is fast and works offline, but requires the remote HEAD ref to exist (set automatically on clone, or manually via `git remote set-head <remote> --auto`).
- The `--base` flag remains as an explicit override for any edge case.
