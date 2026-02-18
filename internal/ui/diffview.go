package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/git"
)

var (
	addedLineStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	removedLineStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	hunkHeaderStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Faint(true)
	lineNoStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(6)
	cursorStyle        = lipgloss.NewStyle().Bold(true)
	cursorLineBg       = lipgloss.Color("236")
	commentMarkerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	visualSelectStyle  = lipgloss.NewStyle().Background(lipgloss.Color("238"))
	sideSeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// emptyStyle is a reusable zero-value style to avoid allocating lipgloss.NewStyle() per call.
var emptyStyle = lipgloss.NewStyle()

// formatLineNo formats a line number right-aligned in a 4-char field followed by a space.
// Returns "     " (5 spaces) for lineNo <= 0.
func formatLineNo(lineNo int) string {
	if lineNo <= 0 {
		return "     "
	}
	var buf [5]byte
	buf[4] = ' '
	// Format the number right-aligned in positions 0-3
	n := lineNo
	i := 3
	for n > 0 && i >= 0 {
		buf[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	// Fill remaining positions with spaces
	for i >= 0 {
		buf[i] = ' '
		i--
	}
	return string(buf[:])
}

// navigateFileMsg signals that the diff viewer has hit a boundary and wants to
// navigate to the next or previous file.
type navigateFileMsg struct {
	direction int // +1 for next, -1 for prev
}

// diffLine is a flattened line for display, which can be a hunk header or a code line.
type diffLine struct {
	isHunkHeader bool
	hunkHeader   string
	line         *git.Line
}

// DiffViewer is a Bubble Tea sub-model for displaying file diffs.
type DiffViewer struct {
	diff             *git.FileDiff
	lines            []diffLine
	cursor           int
	offset           int // scroll offset
	width            int
	height           int
	focused          bool
	commentLines     map[int]bool // lines with comments (by flattened index)
	visualMode       bool
	visualStart      int
	sideBySide       bool
	searchTerm       string
	searchMatches    []int
	searchIdx        int
	pendingBracket   rune // for ]c / [c sequences
	preBracketCursor int  // cursor position before bracket hunk jump
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

// SetCursorToEnd positions the cursor at the last line and scrolls to show it.
func (dv *DiffViewer) SetCursorToEnd() {
	if len(dv.lines) > 0 {
		dv.cursor = len(dv.lines) - 1
		dv.adjustScroll()
	}
}

// SetCommentLines updates which lines have comments.
func (dv *DiffViewer) SetCommentLines(lines map[int]bool) {
	dv.commentLines = lines
}

func (dv *DiffViewer) flattenLines() []diffLine {
	if dv.diff == nil {
		return nil
	}
	// Pre-compute total capacity: one header per hunk plus all lines
	total := 0
	for _, h := range dv.diff.Hunks {
		total += 1 + len(h.Lines)
	}
	result := make([]diffLine, 0, total)
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
				// Restore cursor to pre-bracket position for comment navigation
				dv.cursor = dv.preBracketCursor
				if dv.pendingBracket == ']' {
					dv.jumpToNextComment()
				} else {
					dv.jumpToPrevComment()
				}
				dv.pendingBracket = 0
				return dv, nil
			}
			// Not 'c' — clear pending bracket and fall through to process key normally
			dv.pendingBracket = 0
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
			if !dv.jumpToNextHunk() {
				return dv, func() tea.Msg { return navigateFileMsg{direction: 1} }
			}
		case "{":
			if !dv.jumpToPrevHunk() {
				return dv, func() tea.Msg { return navigateFileMsg{direction: -1} }
			}
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
			dv.preBracketCursor = dv.cursor
			if !dv.jumpToNextChange() {
				dv.pendingBracket = ']'
				return dv, func() tea.Msg { return navigateFileMsg{direction: 1} }
			}
			dv.pendingBracket = ']'
		case "[":
			dv.preBracketCursor = dv.cursor
			if !dv.jumpToPrevChange() {
				dv.pendingBracket = '['
				return dv, func() tea.Msg { return navigateFileMsg{direction: -1} }
			}
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

func (dv *DiffViewer) jumpToNextHunk() bool {
	for i := dv.cursor + 1; i < len(dv.lines); i++ {
		if dv.lines[i].isHunkHeader {
			dv.cursor = i
			dv.adjustScroll()
			return true
		}
	}
	return false
}

func (dv *DiffViewer) jumpToPrevHunk() bool {
	for i := dv.cursor - 1; i >= 0; i-- {
		if dv.lines[i].isHunkHeader {
			dv.cursor = i
			dv.adjustScroll()
			return true
		}
	}
	return false
}

func (dv *DiffViewer) isChangedLine(i int) bool {
	dl := dv.lines[i]
	return dl.line != nil && (dl.line.Type == git.LineAdded || dl.line.Type == git.LineRemoved)
}

func (dv *DiffViewer) jumpToNextChange() bool {
	i := dv.cursor
	// If currently on a changed line, skip past the current change block
	for i < len(dv.lines) && dv.isChangedLine(i) {
		i++
	}
	// Skip past context/headers to find the start of the next change block
	for i < len(dv.lines) && !dv.isChangedLine(i) {
		i++
	}
	if i < len(dv.lines) {
		dv.cursor = i
		dv.adjustScroll()
		return true
	}
	return false
}

func (dv *DiffViewer) jumpToPrevChange() bool {
	i := dv.cursor
	// If currently on a changed line, move back one to leave this block
	if i > 0 && dv.isChangedLine(i) {
		i--
	}
	// Skip past context/headers to find the end of the previous change block
	for i >= 0 && !dv.isChangedLine(i) {
		i--
	}
	// Now on the last line of a change block; skip to its start
	for i > 0 && dv.isChangedLine(i-1) {
		i--
	}
	if i >= 0 && dv.isChangedLine(i) {
		dv.cursor = i
		dv.adjustScroll()
		return true
	}
	return false
}

// View renders the diff.
func (dv DiffViewer) View() string {
	if dv.diff == nil || len(dv.lines) == 0 {
		return "No diff to display. Select a file."
	}

	end := dv.offset + dv.height
	if end > len(dv.lines) {
		end = len(dv.lines)
	}
	visibleLines := end - dv.offset

	var b strings.Builder
	// Estimate ~200 bytes per line for pre-allocation
	b.Grow(visibleLines * 200)

	// Compute visual selection range
	var vStart, vEnd int
	if dv.visualMode {
		vStart, vEnd = dv.VisualRange()
	}

	// Pre-render the cursor arrow once (used at most once per frame)
	cursorArrowStyle := cursorStyle.Background(cursorLineBg)
	cursorBgStyle := emptyStyle.Background(cursorLineBg)

	for i := dv.offset; i < end; i++ {
		dl := dv.lines[i]
		isCursor := i == dv.cursor
		inVisual := dv.visualMode && i >= vStart && i <= vEnd

		var line string
		if dl.isHunkHeader {
			if isCursor {
				line = hunkHeaderStyle.Background(cursorLineBg).Render(dl.hunkHeader)
			} else {
				line = hunkHeaderStyle.Render(dl.hunkHeader)
			}
		} else if dv.sideBySide {
			line = dv.renderSideBySideLine(dl, i, isCursor)
		} else {
			line = dv.renderCodeLine(dl, i, isCursor)
		}

		if inVisual {
			line = visualSelectStyle.Render(line)
		}

		if isCursor {
			arrow := cursorArrowStyle.Render("→ ")
			line = arrow + line
			// Pad to full terminal width
			visible := lipgloss.Width(line)
			if visible < dv.width {
				line += cursorBgStyle.Render(strings.Repeat(" ", dv.width-visible))
			}
		} else if inVisual {
			line = "▎ " + line
		} else {
			line = "  " + line
		}

		b.WriteString(line)
		b.WriteByte('\n')
	}

	return b.String()
}

func (dv DiffViewer) renderCodeLine(dl diffLine, idx int, highlight bool) string {
	l := dl.line

	lnStyle := lineNoStyle
	addStyle := addedLineStyle
	rmStyle := removedLineStyle
	if highlight {
		lnStyle = lnStyle.Background(cursorLineBg)
		addStyle = addStyle.Background(cursorLineBg)
		rmStyle = rmStyle.Background(cursorLineBg)
	}

	oldNo := formatLineNo(l.OldLineNo)
	newNo := formatLineNo(l.NewLineNo)
	gutter := lnStyle.Render(oldNo) + lnStyle.Render(newNo)

	var marker string
	if dv.commentLines[idx] {
		mStyle := commentMarkerStyle
		if highlight {
			mStyle = mStyle.Background(cursorLineBg)
		}
		if highlight {
			bgStyle := emptyStyle.Background(cursorLineBg)
			marker = bgStyle.Render(mStyle.Render("●") + " ")
		} else {
			marker = mStyle.Render("●") + " "
		}
	} else {
		if highlight {
			bgStyle := emptyStyle.Background(cursorLineBg)
			marker = bgStyle.Render("  ")
		} else {
			marker = "  "
		}
	}

	var content string
	switch l.Type {
	case git.LineAdded:
		content = addStyle.Render("+" + l.Content)
	case git.LineRemoved:
		content = rmStyle.Render("-" + l.Content)
	default:
		if highlight {
			bgStyle := emptyStyle.Background(cursorLineBg)
			content = bgStyle.Render(" " + l.Content)
		} else {
			content = " " + l.Content
		}
	}

	return gutter + marker + content
}

// emptyLineNoPad is a pre-computed string of spaces for empty line number gutters.
const emptyLineNoPad = "      " // 6 spaces

func (dv DiffViewer) renderSideBySideLine(dl diffLine, idx int, highlight bool) string {
	l := dl.line
	halfWidth := dv.width / 2

	lnStyle := lineNoStyle
	addStyle := addedLineStyle
	rmStyle := removedLineStyle
	sepStyle := sideSeparatorStyle
	if highlight {
		lnStyle = lnStyle.Background(cursorLineBg)
		addStyle = addStyle.Background(cursorLineBg)
		rmStyle = rmStyle.Background(cursorLineBg)
		sepStyle = sepStyle.Background(cursorLineBg)
	}

	var markerSection string
	if dv.commentLines[idx] {
		mStyle := commentMarkerStyle
		if highlight {
			mStyle = mStyle.Background(cursorLineBg)
			bgStyle := emptyStyle.Background(cursorLineBg)
			markerSection = bgStyle.Render(mStyle.Render("●") + " ")
		} else {
			markerSection = mStyle.Render("●") + " "
		}
	} else {
		if highlight {
			bgStyle := emptyStyle.Background(cursorLineBg)
			markerSection = bgStyle.Render("  ")
		} else {
			markerSection = "  "
		}
	}

	sep := sepStyle.Render("│")

	padToWidth := func(s string, w int) string {
		visible := lipgloss.Width(s)
		if visible < w {
			pad := strings.Repeat(" ", w-visible)
			if highlight {
				bgStyle := emptyStyle.Background(cursorLineBg)
				return s + bgStyle.Render(pad)
			}
			return s + pad
		}
		return s
	}

	var b strings.Builder
	b.Grow(256)

	// renderBg applies background styling only when highlight is active.
	renderBg := func(s string) string {
		if highlight {
			return emptyStyle.Background(cursorLineBg).Render(s)
		}
		return s
	}

	switch l.Type {
	case git.LineRemoved:
		oldNo := formatLineNo(l.OldLineNo)
		leftGutter := lnStyle.Render(oldNo)
		leftContent := rmStyle.Render("-" + l.Content)
		left := padToWidth(leftGutter+leftContent, halfWidth)
		right := padToWidth(renderBg(emptyLineNoPad), halfWidth)
		b.WriteString(left)
		b.WriteString(markerSection)
		b.WriteString(sep)
		b.WriteString(right)

	case git.LineAdded:
		left := padToWidth(renderBg(emptyLineNoPad), halfWidth)
		newNo := formatLineNo(l.NewLineNo)
		rightGutter := lnStyle.Render(newNo)
		rightContent := addStyle.Render("+" + l.Content)
		right := padToWidth(rightGutter+rightContent, halfWidth)
		b.WriteString(left)
		b.WriteString(markerSection)
		b.WriteString(sep)
		b.WriteString(right)

	default: // context
		oldNo := formatLineNo(l.OldLineNo)
		newNo := formatLineNo(l.NewLineNo)
		leftGutter := lnStyle.Render(oldNo)
		left := padToWidth(leftGutter+renderBg(" "+l.Content), halfWidth)
		rightGutter := lnStyle.Render(newNo)
		right := padToWidth(rightGutter+renderBg(" "+l.Content), halfWidth)
		b.WriteString(left)
		b.WriteString(markerSection)
		b.WriteString(sep)
		b.WriteString(right)
	}

	return b.String()
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
