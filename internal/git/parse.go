package git

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	diffHeaderRe = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
)

// ParseDiff parses unified diff output into a slice of FileDiff values.
func ParseDiff(raw string) ([]FileDiff, error) {
	if raw == "" {
		return nil, nil
	}

	var diffs []FileDiff
	lines := strings.Split(raw, "\n")

	var current *FileDiff

	for i := range lines {
		line := lines[i]

		// Match diff --git header to start a new file diff.
		if m := diffHeaderRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				diffs = append(diffs, *current)
			}
			current = &FileDiff{
				Path: m[2],
			}
			continue
		}

		// Skip index, mode, and --- / +++ headers.
		if strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "old mode ") ||
			strings.HasPrefix(line, "new mode ") ||
			strings.HasPrefix(line, "new file mode ") ||
			strings.HasPrefix(line, "deleted file mode ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") {
			continue
		}

		// Match hunk header.
		if m := hunkHeaderRe.FindStringSubmatch(line); m != nil {
			if current == nil {
				continue
			}
			h := Hunk{
				OldStart: atoi(m[1]),
				OldCount: atoiDefault(m[2], 1),
				NewStart: atoi(m[3]),
				NewCount: atoiDefault(m[4], 1),
				Header:   line,
			}
			current.Hunks = append(current.Hunks, h)
			continue
		}

		// Skip "\ No newline at end of file" lines.
		if strings.HasPrefix(line, `\ `) {
			continue
		}

		// Parse content lines within a hunk.
		if current == nil || len(current.Hunks) == 0 {
			continue
		}

		hunk := &current.Hunks[len(current.Hunks)-1]

		switch {
		case strings.HasPrefix(line, "+"):
			hunk.Lines = append(hunk.Lines, Line{
				Content: line[1:],
				Type:    LineAdded,
			})
		case strings.HasPrefix(line, "-"):
			hunk.Lines = append(hunk.Lines, Line{
				Content: line[1:],
				Type:    LineRemoved,
			})
		case strings.HasPrefix(line, " "):
			hunk.Lines = append(hunk.Lines, Line{
				Content: line[1:],
				Type:    LineContext,
			})
		case line == "" && !hunkComplete(hunk):
			// Empty lines within a hunk represent blank context lines.
			// Only include them if the hunk still expects more lines.
			hunk.Lines = append(hunk.Lines, Line{
				Content: "",
				Type:    LineContext,
			})
		}
	}

	if current != nil {
		diffs = append(diffs, *current)
	}

	// Assign line numbers to all hunks.
	for i := range diffs {
		for j := range diffs[i].Hunks {
			assignLineNumbers(&diffs[i].Hunks[j])
		}
	}

	return diffs, nil
}

// hunkComplete returns true if the hunk has consumed all expected old and new lines.
func hunkComplete(h *Hunk) bool {
	var oldConsumed, newConsumed int
	for _, l := range h.Lines {
		switch l.Type {
		case LineContext:
			oldConsumed++
			newConsumed++
		case LineAdded:
			newConsumed++
		case LineRemoved:
			oldConsumed++
		}
	}
	return oldConsumed >= h.OldCount && newConsumed >= h.NewCount
}

// assignLineNumbers fills in OldLineNo and NewLineNo for each line in a hunk.
func assignLineNumbers(h *Hunk) {
	oldNo := h.OldStart
	newNo := h.NewStart

	for i := range h.Lines {
		switch h.Lines[i].Type {
		case LineContext:
			h.Lines[i].OldLineNo = oldNo
			h.Lines[i].NewLineNo = newNo
			oldNo++
			newNo++
		case LineAdded:
			h.Lines[i].NewLineNo = newNo
			newNo++
		case LineRemoved:
			h.Lines[i].OldLineNo = oldNo
			oldNo++
		}
	}
}

// ParseNameStatus parses git diff --name-status output into a slice of ChangedFile values.
func ParseNameStatus(raw string) []ChangedFile {
	if raw == "" {
		return nil
	}

	var files []ChangedFile
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		files = append(files, ChangedFile{
			Status: parts[0],
			Path:   parts[1],
		})
	}
	return files
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	return atoi(s)
}
