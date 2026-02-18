# Quality Pass Design

## Goal

Review and optimize the entire revui codebase for idiomatic Go, performance, and cleanliness before pushing local changes.

## Scope

Conservative — fix concrete issues, add benchmarks, optimize hot paths, idiomatic cleanup. No structural refactors.

## Tasks

### 1. Idiomatic Cleanup

- `internal/ui/filelist.go`: Replace `s += ...` string concatenation loop in `View()` with `strings.Builder`
- `internal/comment/comment.go`: Replace slice-based `Get()`/`Delete()` with map keying for O(1) lookups
- General review by Go expert agent for non-idiomatic patterns across all packages

### 2. Benchmarks

Add benchmark functions for hot paths:

| Benchmark | File | Function |
|-----------|------|----------|
| `BenchmarkParseDiff` | `internal/git/parse_test.go` | `ParseDiff()` |
| `BenchmarkRenderCodeLine` | `internal/ui/diffview_test.go` | `renderCodeLine()` |
| `BenchmarkRenderSideBySideLine` | `internal/ui/diffview_test.go` | `renderSideBySideLine()` |
| `BenchmarkFlattenLines` | `internal/ui/diffview_test.go` | `flattenLines()` |
| `BenchmarkBuildSideBySidePairs` | `internal/ui/sidebyside_test.go` | `BuildSideBySidePairs()` |
| `BenchmarkCommentFormat` | `internal/comment/format_test.go` | `Format()` |

### 3. Optimize Based on Benchmark Results

Profile allocations (`-benchmem`) and apply easy wins: pre-allocated slices, reduced string copies, buffer reuse.

### 4. Go Expert Review

Full codebase review by Go expert agent for idiomatic issues, performance, and correctness.

### 5. Final Pass

- Run `go fix ./...`
- Run `go vet ./...`
- Verify all tests and benchmarks pass
