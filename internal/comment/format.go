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
	// Estimate capacity: header + per-file sections
	b.Grow(256 * len(comments))
	b.WriteString("## Code Review Comments\n\n")

	for i, file := range fileOrder {
		b.WriteString("### ")
		b.WriteString(file)
		b.WriteString("\n\n")

		for _, c := range grouped[file] {
			b.WriteString("**")
			writeLineInfo(&b, c)
			b.WriteString(":**\n")

			if c.CodeSnippet != "" {
				ext := fileExtension(file)
				b.WriteString("```")
				b.WriteString(ext)
				b.WriteByte('\n')
				b.WriteString(c.CodeSnippet)
				b.WriteString("\n```\n")
			}

			b.WriteString("**Comment:** ")
			b.WriteString(c.Body)
			b.WriteString("\n\n")
		}

		if i < len(fileOrder)-1 {
			b.WriteString("---\n\n")
		}
	}

	return b.String()
}

// writeLineInfo writes the line info directly to a builder, avoiding intermediate string allocation.
func writeLineInfo(b *strings.Builder, c Comment) {
	lineType := c.LineType.String()

	if c.StartLine == c.EndLine || c.EndLine == 0 {
		b.WriteString("Line ")
		b.WriteString(strconv.Itoa(c.StartLine))
	} else {
		b.WriteString("Lines ")
		b.WriteString(strconv.Itoa(c.StartLine))
		b.WriteByte('-')
		b.WriteString(strconv.Itoa(c.EndLine))
	}

	if lineType != "context" {
		b.WriteString(" (")
		b.WriteString(lineType)
		b.WriteByte(')')
	}
}

func fileExtension(path string) string {
	idx := strings.LastIndexByte(path, '.')
	if idx >= 0 {
		return path[idx+1:]
	}
	return ""
}
