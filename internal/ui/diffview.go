package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/git"
	"github.com/deparker/revui/internal/syntax"
)

var (
	addedLineStyle   = lipgloss.NewStyle().Background(lipgloss.Color("22"))
	removedLineStyle = lipgloss.NewStyle().Background(lipgloss.Color("52"))
	contextLineStyle = lipgloss.NewStyle()
	hunkHeaderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Faint(true)
	lineNoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(6)
	cursorStyle      = lipgloss.NewStyle().Bold(true)
	commentMarker    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("●")
	visualSelectStyle  = lipgloss.NewStyle().Background(lipgloss.Color("238"))
	sideSeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// diffLine is a flattened line for display, which can be a hunk header or a code line.
type diffLine struct {
	isHunkHeader bool
	hunkHeader   string
	line         *git.Line
}

// DiffViewer is a Bubble Tea sub-model for displaying file diffs.
type DiffViewer struct {
	diff               *git.FileDiff
	lines              []diffLine
	cursor             int
	offset             int // scroll offset
	width              int
	height             int
	focused            bool
	commentLines       map[int]bool // lines with comments (by flattened index)
	highlighter        *syntax.Highlighter
	highlightEnabled   bool
	visualMode         bool
	visualStart        int
	sideBySide         bool
	searchTerm         string
	searchMatches      []int
	searchIdx          int
	pendingBracket     rune // for ]c / [c sequences
}

// NewDiffViewer creates a new diff viewer.
func NewDiffViewer(width, height int) DiffViewer {
	return DiffViewer{
		width:            width,
		height:           height,
		commentLines:     make(map[int]bool),
		highlighter:      syntax.NewHighlighter(),
		highlightEnabled: true,
	}
}

// EnableSyntaxHighlighting enables or disables syntax highlighting.
func (dv *DiffViewer) EnableSyntaxHighlighting(enabled bool) {
	dv.highlightEnabled = enabled
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
		key := msg.String()

		// Handle pending bracket sequences (]c / [c)
		if dv.pendingBracket != 0 {
			if key == "c" {
				if dv.pendingBracket == ']' {
					dv.jumpToNextComment()
				} else {
					dv.jumpToPrevComment()
				}
			}
			dv.pendingBracket = 0
			return dv, nil
		}

		switch key {
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
		case "v":
			if dv.visualMode {
				dv.visualMode = false
			} else {
				dv.visualMode = true
				dv.visualStart = dv.cursor
			}
		case "esc":
			dv.visualMode = false
		case "tab":
			dv.sideBySide = !dv.sideBySide
		case "]":
			dv.pendingBracket = ']'
		case "[":
			dv.pendingBracket = '['
		case "n":
			dv.jumpToNextSearch()
		case "N":
			dv.jumpToPrevSearch()
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

	// Compute visual selection range
	var vStart, vEnd int
	if dv.visualMode {
		vStart, vEnd = dv.VisualRange()
	}

	for i := dv.offset; i < end; i++ {
		dl := dv.lines[i]
		isCursor := i == dv.cursor
		inVisual := dv.visualMode && i >= vStart && i <= vEnd

		var line string
		if dl.isHunkHeader {
			line = hunkHeaderStyle.Render(dl.hunkHeader)
		} else if dv.sideBySide {
			line = dv.renderSideBySideLine(dl, i)
		} else {
			line = dv.renderCodeLine(dl, i)
		}

		if inVisual {
			line = visualSelectStyle.Render(line)
		}

		if isCursor {
			line = cursorStyle.Render("→ ") + line
		} else if inVisual {
			line = "▎ " + line
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

	text := l.Content
	if dv.highlightEnabled && dv.diff != nil {
		text = dv.highlighter.HighlightLine(dv.diff.Path, l.Content)
	}

	var content string
	switch l.Type {
	case git.LineAdded:
		content = addedLineStyle.Render("+") + text
	case git.LineRemoved:
		content = removedLineStyle.Render("-") + text
	default:
		content = " " + text
	}

	return gutter + marker + " " + content
}

func (dv DiffViewer) renderSideBySideLine(dl diffLine, idx int) string {
	l := dl.line
	halfWidth := dv.width / 2

	marker := " "
	if dv.commentLines[idx] {
		marker = commentMarker
	}

	text := l.Content
	if dv.highlightEnabled && dv.diff != nil {
		text = dv.highlighter.HighlightLine(dv.diff.Path, l.Content)
	}

	sep := sideSeparatorStyle.Render("│")
	lineNoWidth := 6
	emptyLineNo := strings.Repeat(" ", lineNoWidth)

	padToWidth := func(s string, w int) string {
		visible := lipgloss.Width(s)
		if visible < w {
			return s + strings.Repeat(" ", w-visible)
		}
		return s
	}

	switch l.Type {
	case git.LineRemoved:
		oldNo := fmt.Sprintf("%4d ", l.OldLineNo)
		leftGutter := lineNoStyle.Render(oldNo)
		leftContent := removedLineStyle.Render("-") + text
		left := padToWidth(leftGutter+leftContent, halfWidth)
		right := padToWidth(emptyLineNo, halfWidth)
		return left + marker + sep + right

	case git.LineAdded:
		left := padToWidth(emptyLineNo, halfWidth)
		newNo := fmt.Sprintf("%4d ", l.NewLineNo)
		rightGutter := lineNoStyle.Render(newNo)
		rightContent := addedLineStyle.Render("+") + text
		right := padToWidth(rightGutter+rightContent, halfWidth)
		return left + marker + sep + right

	default: // context
		oldNo := fmt.Sprintf("%4d ", l.OldLineNo)
		newNo := fmt.Sprintf("%4d ", l.NewLineNo)
		leftGutter := lineNoStyle.Render(oldNo)
		left := padToWidth(leftGutter+" "+text, halfWidth)
		rightGutter := lineNoStyle.Render(newNo)
		right := padToWidth(rightGutter+" "+text, halfWidth)
		return left + marker + sep + right
	}
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

// lineAt returns the diffLine at the given index, or nil if out of bounds.
func (dv DiffViewer) lineAt(idx int) *diffLine {
	if idx >= 0 && idx < len(dv.lines) {
		return &dv.lines[idx]
	}
	return nil
}

// LineNoAt returns the relevant line number at the given flattened index.
func (dv DiffViewer) LineNoAt(idx int) int {
	dl := dv.lineAt(idx)
	if dl == nil || dl.line == nil {
		return 0
	}
	if dl.line.Type == git.LineRemoved {
		return dl.line.OldLineNo
	}
	return dl.line.NewLineNo
}

// SnippetRange returns the code snippet text for lines between start and end indices (inclusive).
func (dv DiffViewer) SnippetRange(start, end int) string {
	var lines []string
	for i := start; i <= end; i++ {
		dl := dv.lineAt(i)
		if dl != nil && dl.line != nil {
			lines = append(lines, dl.line.Content)
		}
	}
	return strings.Join(lines, "\n")
}

// ExitVisualMode exits visual mode.
func (dv *DiffViewer) ExitVisualMode() {
	dv.visualMode = false
}

// InVisualMode returns whether visual mode is active.
func (dv DiffViewer) InVisualMode() bool {
	return dv.visualMode
}

// VisualRange returns the start and end of the visual selection.
func (dv DiffViewer) VisualRange() (int, int) {
	start, end := dv.visualStart, dv.cursor
	if start > end {
		start, end = end, start
	}
	return start, end
}

// IsSideBySide returns whether side-by-side mode is active.
func (dv DiffViewer) IsSideBySide() bool {
	return dv.sideBySide
}

// SetSearch sets the search term and computes matches.
func (dv *DiffViewer) SetSearch(term string) {
	dv.searchTerm = term
	dv.searchMatches = nil
	dv.searchIdx = 0
	if term == "" {
		return
	}
	for i, dl := range dv.lines {
		if dl.line != nil && strings.Contains(dl.line.Content, term) {
			dv.searchMatches = append(dv.searchMatches, i)
		}
	}
}

// SearchMatches returns the indices of lines matching the search term.
func (dv DiffViewer) SearchMatches() []int {
	return dv.searchMatches
}

func (dv *DiffViewer) jumpToNextSearch() {
	if len(dv.searchMatches) == 0 {
		return
	}
	// Find next match after cursor
	for _, idx := range dv.searchMatches {
		if idx > dv.cursor {
			dv.cursor = idx
			dv.adjustScroll()
			return
		}
	}
	// Wrap around
	dv.cursor = dv.searchMatches[0]
	dv.adjustScroll()
}

func (dv *DiffViewer) jumpToPrevSearch() {
	if len(dv.searchMatches) == 0 {
		return
	}
	// Find previous match before cursor
	for i := len(dv.searchMatches) - 1; i >= 0; i-- {
		if dv.searchMatches[i] < dv.cursor {
			dv.cursor = dv.searchMatches[i]
			dv.adjustScroll()
			return
		}
	}
	// Wrap around
	dv.cursor = dv.searchMatches[len(dv.searchMatches)-1]
	dv.adjustScroll()
}

func (dv *DiffViewer) jumpToNextComment() {
	for i := dv.cursor + 1; i < len(dv.lines); i++ {
		if dv.commentLines[i] {
			dv.cursor = i
			dv.adjustScroll()
			return
		}
	}
}

func (dv *DiffViewer) jumpToPrevComment() {
	for i := dv.cursor - 1; i >= 0; i-- {
		if dv.commentLines[i] {
			dv.cursor = i
			dv.adjustScroll()
			return
		}
	}
}
