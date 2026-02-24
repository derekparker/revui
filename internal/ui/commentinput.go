package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/git"
)

var commentInputStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("3")).
	Padding(0, 1)

// CommentSubmitMsg is sent when the user submits a comment.
type CommentSubmitMsg struct {
	FilePath    string
	LineNo      int
	EndLineNo   int
	Body        string
	LineType    git.LineType
}

// CommentCancelMsg is sent when the user cancels comment input.
type CommentCancelMsg struct{}

// CommentInput is a sub-model for entering review comments.
type CommentInput struct {
	input       textinput.Model
	active      bool
	filePath    string
	lineNo      int
	endLineNo   int
	lineType    git.LineType
	width       int
}

// NewCommentInput creates a new comment input component.
func NewCommentInput(width int) CommentInput {
	ti := textinput.New()
	ti.Placeholder = "Enter comment..."
	ti.CharLimit = 500
	ti.Width = width - 6

	return CommentInput{
		input: ti,
		width: width,
	}
}

// Activate shows the input and optionally pre-fills with an existing comment.
func (ci *CommentInput) Activate(filePath string, lineNo, endLineNo int, lineType git.LineType, existing string) {
	ci.active = true
	ci.filePath = filePath
	ci.lineNo = lineNo
	ci.endLineNo = endLineNo
	ci.lineType = lineType
	ci.input.SetValue(existing)
	ci.input.Focus()
}

// Init returns the text input blink command.
func (ci CommentInput) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles key messages.
func (ci CommentInput) Update(msg tea.Msg) (CommentInput, tea.Cmd) {
	if !ci.active {
		return ci, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			ci.active = false
			ci.input.Blur()
			return ci, func() tea.Msg { return CommentCancelMsg{} }
		case tea.KeyEnter:
			body := ci.input.Value()
			ci.active = false
			ci.input.Blur()
			if body == "" {
				return ci, func() tea.Msg { return CommentCancelMsg{} }
			}
			submitMsg := CommentSubmitMsg{
				FilePath:  ci.filePath,
				LineNo:    ci.lineNo,
				EndLineNo: ci.endLineNo,
				Body:      body,
				LineType:  ci.lineType,
			}
			return ci, func() tea.Msg { return submitMsg }
		}
	}

	var cmd tea.Cmd
	ci.input, cmd = ci.input.Update(msg)
	return ci, cmd
}

// View renders the comment input.
func (ci CommentInput) View() string {
	if !ci.active {
		return ""
	}
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Comment: ")
	return commentInputStyle.Render(label + ci.input.View())
}

// Active returns whether the input is currently shown.
func (ci CommentInput) Active() bool {
	return ci.active
}

// FilePath returns the file being commented on.
func (ci CommentInput) FilePath() string {
	return ci.filePath
}

// LineNo returns the line being commented on.
func (ci CommentInput) LineNo() int {
	return ci.lineNo
}

// Value returns the current input text.
func (ci CommentInput) Value() string {
	return ci.input.Value()
}

// SetWidth updates the width.
func (ci *CommentInput) SetWidth(width int) {
	ci.width = width
	ci.input.Width = width - 6
}
