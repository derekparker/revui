package ui

import "github.com/deparker/revui/internal/git"

// LinePair represents a paired line for side-by-side display.
type LinePair struct {
	Left  *git.Line
	Right *git.Line
}

// BuildSideBySidePairs pairs removed/added lines side-by-side and maps context lines to both sides.
func BuildSideBySidePairs(lines []git.Line) []LinePair {
	var pairs []LinePair
	var removed []*git.Line

	for i := range lines {
		l := &lines[i]
		switch l.Type {
		case git.LineRemoved:
			removed = append(removed, l)
		case git.LineAdded:
			if len(removed) > 0 {
				// Pair with a pending removed line
				pairs = append(pairs, LinePair{Left: removed[0], Right: l})
				removed = removed[1:]
			} else {
				pairs = append(pairs, LinePair{Right: l})
			}
		case git.LineContext:
			// Flush remaining removed lines
			for _, r := range removed {
				pairs = append(pairs, LinePair{Left: r})
			}
			removed = nil
			pairs = append(pairs, LinePair{Left: l, Right: l})
		}
	}

	// Flush any remaining removed lines
	for _, r := range removed {
		pairs = append(pairs, LinePair{Left: r})
	}

	return pairs
}
