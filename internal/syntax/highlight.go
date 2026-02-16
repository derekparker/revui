package syntax

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlighter provides syntax highlighting for code lines.
type Highlighter struct {
	style     *chroma.Style
	formatter chroma.Formatter
}

// NewHighlighter creates a highlighter with a terminal-friendly dark theme.
func NewHighlighter() *Highlighter {
	return &Highlighter{
		style:     styles.Get("monokai"),
		formatter: formatters.TTY256,
	}
}

// HighlightLine applies syntax highlighting to a single line of code.
func (h *Highlighter) HighlightLine(filename, line string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, line)
	if err != nil {
		return line
	}

	var buf bytes.Buffer
	if err := h.formatter.Format(&buf, h.style, iterator); err != nil {
		return line
	}

	return strings.TrimRight(buf.String(), "\n")
}

// ExtensionFromPath returns the file extension for lexer matching.
func ExtensionFromPath(path string) string {
	return filepath.Ext(path)
}
