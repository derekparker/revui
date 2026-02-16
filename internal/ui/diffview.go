package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/git"
)

var (
	addedLineStyle   = lipgloss.NewStyle().Background(lipgloss.Color("22"))
	removedLineStyle = lipgloss.NewStyle().Background(lipgloss.Color("52"))
	contextLineStyle = lipgloss.NewStyle()
	hunkHeaderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Faint(true)
	lineNoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(6)
	cursorStyle      = lipgloss.NewStyle().Bold(true)
	commentMarker    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("●")
)

// diffLine is a flattened line for display, which can be a hunk header or a code line.
type diffLine struct {
	isHunkHeader bool
	hunkHeader   string
	line         *git.Line
}

// DiffViewer is a Bubble Tea sub-model for displaying file diffs.
type DiffViewer struct {
	diff         *git.FileDiff
	lines        []diffLine
	cursor       int
	offset       int // scroll offset
	width        int
	height       int
	focused      bool
	commentLines map[int]bool // lines with comments (by flattened index)
}

// NewDiffViewer creates a new diff viewer.
func NewDiffViewer(width, height int) DiffViewer {
	return DiffViewer{
		width:        width,
		height:       height,
		commentLines: make(map[int]bool),
	}
}

// SetDiff sets the diff content to display.
func (dv *DiffViewer) SetDiff(fd *git.FileDiff) {
	dv.diff = fd
	dv.cursor = 0
	dv.offset = 0
	dv.lines = dv.flattenLines()
}

// SetCommentLines updates which lines have comments.
func (dv *DiffViewer) SetCommentLines(lines map[int]bool) {
	dv.commentLines = lines
}

func (dv *DiffViewer) flattenLines() []diffLine {
	if dv.diff == nil {
		return nil
	}
	var result []diffLine
	for _, h := range dv.diff.Hunks {
		result = append(result, diffLine{
			isHunkHeader: true,
			hunkHeader:   h.Header,
		})
		for i := range h.Lines {
			result = append(result, diffLine{
				line: &h.Lines[i],
			})
		}
	}
	return result
}

// Init returns no initial command.
func (dv DiffViewer) Init() tea.Cmd {
	return nil
}

// Update handles key messages for vim-style navigation.
func (dv DiffViewer) Update(msg tea.Msg) (DiffViewer, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if dv.cursor < len(dv.lines)-1 {
				dv.cursor++
				dv.adjustScroll()
			}
		case "k", "up":
			if dv.cursor > 0 {
				dv.cursor--
				dv.adjustScroll()
			}
		case "G":
			dv.cursor = len(dv.lines) - 1
			dv.adjustScroll()
		case "g":
			dv.cursor = 0
			dv.offset = 0
		case "ctrl+d":
			dv.cursor += dv.height / 2
			if dv.cursor >= len(dv.lines) {
				dv.cursor = len(dv.lines) - 1
			}
			dv.adjustScroll()
		case "ctrl+u":
			dv.cursor -= dv.height / 2
			if dv.cursor < 0 {
				dv.cursor = 0
			}
			dv.adjustScroll()
		case "ctrl+f":
			dv.cursor += dv.height
			if dv.cursor >= len(dv.lines) {
				dv.cursor = len(dv.lines) - 1
			}
			dv.adjustScroll()
		case "ctrl+b":
			dv.cursor -= dv.height
			if dv.cursor < 0 {
				dv.cursor = 0
			}
			dv.adjustScroll()
		case "}":
			dv.jumpToNextHunk()
		case "{":
			dv.jumpToPrevHunk()
		}
	}
	return dv, nil
}

func (dv *DiffViewer) adjustScroll() {
	if dv.cursor < dv.offset {
		dv.offset = dv.cursor
	}
	if dv.cursor >= dv.offset+dv.height {
		dv.offset = dv.cursor - dv.height + 1
	}
}

func (dv *DiffViewer) jumpToNextHunk() {
	for i := dv.cursor + 1; i < len(dv.lines); i++ {
		if dv.lines[i].isHunkHeader {
			dv.cursor = i
			dv.adjustScroll()
			return
		}
	}
}

func (dv *DiffViewer) jumpToPrevHunk() {
	for i := dv.cursor - 1; i >= 0; i-- {
		if dv.lines[i].isHunkHeader {
			dv.cursor = i
			dv.adjustScroll()
			return
		}
	}
}

// View renders the diff.
func (dv DiffViewer) View() string {
	if dv.diff == nil || len(dv.lines) == 0 {
		return "No diff to display. Select a file."
	}

	var b strings.Builder

	end := dv.offset + dv.height
	if end > len(dv.lines) {
		end = len(dv.lines)
	}

	for i := dv.offset; i < end; i++ {
		dl := dv.lines[i]
		isCursor := i == dv.cursor

		var line string
		if dl.isHunkHeader {
			line = hunkHeaderStyle.Render(dl.hunkHeader)
		} else {
			line = dv.renderCodeLine(dl, i)
		}

		if isCursor {
			line = cursorStyle.Render("→ ") + line
		} else {
			line = "  " + line
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func (dv DiffViewer) renderCodeLine(dl diffLine, idx int) string {
	l := dl.line
	oldNo := "     "
	newNo := "     "
	if l.OldLineNo > 0 {
		oldNo = fmt.Sprintf("%4d ", l.OldLineNo)
	}
	if l.NewLineNo > 0 {
		newNo = fmt.Sprintf("%4d ", l.NewLineNo)
	}

	gutter := lineNoStyle.Render(oldNo) + lineNoStyle.Render(newNo)

	marker := " "
	if dv.commentLines[idx] {
		marker = commentMarker
	}

	var content string
	switch l.Type {
	case git.LineAdded:
		content = addedLineStyle.Render("+" + l.Content)
	case git.LineRemoved:
		content = removedLineStyle.Render("-" + l.Content)
	default:
		content = contextLineStyle.Render(" " + l.Content)
	}

	return gutter + marker + " " + content
}

// CursorLine returns the current cursor position.
func (dv DiffViewer) CursorLine() int {
	return dv.cursor
}

// CurrentLine returns the git.Line at the cursor, or nil if on a hunk header.
func (dv DiffViewer) CurrentLine() *git.Line {
	if dv.cursor >= 0 && dv.cursor < len(dv.lines) {
		return dv.lines[dv.cursor].line
	}
	return nil
}

// CurrentLineNo returns the relevant line number for commenting (new line for added/context, old for removed).
func (dv DiffViewer) CurrentLineNo() int {
	l := dv.CurrentLine()
	if l == nil {
		return 0
	}
	if l.Type == git.LineRemoved {
		return l.OldLineNo
	}
	return l.NewLineNo
}

// SetFocused sets whether this component has focus.
func (dv *DiffViewer) SetFocused(focused bool) {
	dv.focused = focused
}

// SetSize updates the dimensions.
func (dv *DiffViewer) SetSize(width, height int) {
	dv.width = width
	dv.height = height
}

// TotalLines returns the total number of flattened lines.
func (dv DiffViewer) TotalLines() int {
	return len(dv.lines)
}
