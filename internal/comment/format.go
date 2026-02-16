package comment

import (
	"fmt"
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
	b.WriteString("## Code Review Comments\n\n")

	for i, file := range fileOrder {
		b.WriteString(fmt.Sprintf("### %s\n\n", file))

		for _, c := range grouped[file] {
			lineInfo := formatLineInfo(c)
			b.WriteString(fmt.Sprintf("**%s:**\n", lineInfo))

			if c.CodeSnippet != "" {
				ext := fileExtension(file)
				b.WriteString(fmt.Sprintf("```%s\n%s\n```\n", ext, c.CodeSnippet))
			}

			b.WriteString(fmt.Sprintf("**Comment:** %s\n\n", c.Body))
		}

		if i < len(fileOrder)-1 {
			b.WriteString("---\n\n")
		}
	}

	return b.String()
}

func formatLineInfo(c Comment) string {
	lineType := ""
	if c.LineType.String() != "context" {
		lineType = fmt.Sprintf(" (%s)", c.LineType.String())
	}
	if c.StartLine == c.EndLine || c.EndLine == 0 {
		return fmt.Sprintf("Line %d%s", c.StartLine, lineType)
	}
	return fmt.Sprintf("Lines %d-%d%s", c.StartLine, c.EndLine, lineType)
}

func fileExtension(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}
