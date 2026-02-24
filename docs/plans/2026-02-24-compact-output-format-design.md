# Compact Output Format Design

## Problem

The review output (markdown copied to clipboard / sent to Claude Code) is verbose. Decorative markdown headers, bold labels, fenced code blocks with language hints, and separator lines consume tokens unnecessarily when the primary consumer is an LLM.

## Decision

- Target: Claude Code via tmux delivery
- Priority: Maximum token savings over human readability
- Code snippets: Remove entirely (Claude Code can read files via @-refs)
- Format: Minimal markdown (grouped by file, bullet-point comments)

## Output Format

### Before (~45 tokens per comment)

```markdown
## Code Review Comments

### main.go

**Line 10 (added):**
```go
func doThing() {
```
**Comment:** This needs error handling.

---
```

### After (~15 tokens per comment)

```
main.go
- L10 (added): This needs error handling.
- L25: Consider renaming this variable.

util.go
- L5-8 (removed): Why was this removed?
```

### Format Rules

- File path on its own line (no markdown decoration)
- Each comment: `- L{start}` or `- L{start}-{end}`, then ` ({type})` only for added/removed (omit for context), then `: {body}`
- Blank line between file groups
- No `##`/`###` headers, no `**bold**`, no `---` separators, no fenced code blocks
- No code snippets

## Scope

### Modified Files

| File | Change |
|------|--------|
| `internal/comment/comment.go` | Remove `CodeSnippet` field from `Comment` struct |
| `internal/comment/format.go` | Rewrite `Format()`, simplify `writeLineInfo()`, remove `fileExtension()` |
| `internal/comment/format_test.go` | Update test expectations, remove `CodeSnippet` from test data |
| `internal/ui/commentinput.go` | Remove `CodeSnippet` from `commentInputMsg`, remove `codeSnippet` field, remove snippet param from `Activate()` |
| `internal/ui/diffview.go` | Remove `SnippetRange()` method |
| `internal/ui/root.go` | Remove `CodeSnippet` from comment creation, remove `SnippetRange()` call |

### Not Modified

- `internal/output/` - Delivery is format-agnostic
- `internal/git/` - No changes to diff parsing
