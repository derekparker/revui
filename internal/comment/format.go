package comment

import (
	"strconv"
	"strings"
)

func Format(comments []Comment) string {
	if len(comments) == 0 {
		return ""
	}

	grouped := make(map[string][]Comment)
	var fileOrder []string
	seen := make(map[string]bool)
	for _, c := range comments {
		if !seen[c.FilePath] {
			fileOrder = append(fileOrder, c.FilePath)
			seen[c.FilePath] = true
		}
		grouped[c.FilePath] = append(grouped[c.FilePath], c)
	}

	var b strings.Builder
	b.Grow(64 * len(comments))

	for i, file := range fileOrder {
		b.WriteString(file)
		b.WriteByte('\n')

		for _, c := range grouped[file] {
			b.WriteString("- ")
			writeLineInfo(&b, c)
			b.WriteString(": ")
			b.WriteString(c.Body)
			b.WriteByte('\n')
		}

		if i < len(fileOrder)-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// writeLineInfo writes the line info directly to a builder, avoiding intermediate string allocation.
func writeLineInfo(b *strings.Builder, c Comment) {
	b.WriteByte('L')
	b.WriteString(strconv.Itoa(c.StartLine))

	if c.EndLine != 0 && c.EndLine != c.StartLine {
		b.WriteByte('-')
		b.WriteString(strconv.Itoa(c.EndLine))
	}

	lineType := c.LineType.String()
	if lineType != "context" {
		b.WriteString(" (")
		b.WriteString(lineType)
		b.WriteByte(')')
	}
}
