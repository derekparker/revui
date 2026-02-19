package git

// LineType represents the type of a diff line.
type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
)

func (lt LineType) String() string {
	switch lt {
	case LineAdded:
		return "added"
	case LineRemoved:
		return "removed"
	default:
		return "context"
	}
}

// FileStatusString returns a human-readable string for a git status code.
func FileStatusString(status string) string {
	switch status {
	case "A":
		return "added"
	case "M":
		return "modified"
	case "D":
		return "deleted"
	case "R":
		return "renamed"
	case "B":
		return "binary"
	default:
		return "unknown"
	}
}

// Line represents a single line in a diff.
type Line struct {
	Content   string
	Type      LineType
	OldLineNo int
	NewLineNo int
}

// Hunk represents a contiguous section of a diff.
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Header   string
	Lines    []Line
}

// FileDiff represents the diff for a single file.
type FileDiff struct {
	Path   string
	Status string // A, M, D, R, B
	Hunks  []Hunk
}

// ChangedFile represents a file that changed between two refs.
type ChangedFile struct {
	Path   string
	Status string
}
