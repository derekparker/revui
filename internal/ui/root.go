package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/comment"
	"github.com/deparker/revui/internal/git"
)

type focusArea int

const (
	focusFileList focusArea = iota
	focusDiffViewer
	focusCommentInput
)

// GitRunner is the interface for git operations, enabling testing with mocks.
type GitRunner interface {
	ChangedFiles(base string) ([]git.ChangedFile, error)
	FileDiff(base, path string) (*git.FileDiff, error)
	CurrentBranch() (string, error)
}

// finishMsg signals the review is done and comments should be copied.
type finishMsg struct{}

// RootModel is the top-level Bubble Tea model.
type RootModel struct {
	git           GitRunner
	base          string
	branch        string
	files         []git.ChangedFile
	fileList      FileList
	diffViewer    DiffViewer
	commentInput  CommentInput
	comments      *comment.Store
	focus         focusArea
	width         int
	height        int
	err           error
	quitting      bool
	finished      bool
	output        string // formatted comments for clipboard
	fileListWidth int
	pendingZ      bool
	showHelp      bool
	searchInput   textinput.Model
	searching     bool
}

// NewRootModel creates the root model with the given git runner and base branch.
func NewRootModel(gitRunner GitRunner, base string, width, height int) RootModel {
	fileListWidth := 30

	files, err := gitRunner.ChangedFiles(base)
	if err != nil {
		return RootModel{err: err}
	}

	branch, _ := gitRunner.CurrentBranch()

	fl := NewFileList(files, fileListWidth, height-2)
	dv := NewDiffViewer(width-fileListWidth-3, height-2)
	ci := NewCommentInput(width)

	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 100
	si.Width = width - 10

	// Load the first file's diff if available
	if len(files) > 0 {
		if fd, err := gitRunner.FileDiff(base, files[0].Path); err == nil {
			dv.SetDiff(fd)
		}
	}

	return RootModel{
		git:           gitRunner,
		base:          base,
		branch:        branch,
		files:         files,
		fileList:      fl,
		diffViewer:    dv,
		commentInput:  ci,
		searchInput:   si,
		comments:      comment.NewStore(),
		focus:         focusFileList,
		width:         width,
		height:        height,
		fileListWidth: fileListWidth,
	}
}

// Init returns the initial command.
func (m RootModel) Init() tea.Cmd {
	return nil
}

// Update handles all messages. Returns tea.Model for the interface.
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.fileList.SetSize(m.fileListWidth, m.height-2)
		m.diffViewer.SetSize(m.width-m.fileListWidth-3, m.height-2)
		m.commentInput.SetWidth(m.width)
		return m, nil

	case CommentSubmitMsg:
		line := m.diffViewer.CurrentLine()
		lineType := git.LineContext
		snippet := ""
		if line != nil {
			lineType = line.Type
			snippet = line.Content
		}
		m.comments.Add(comment.Comment{
			FilePath:    msg.FilePath,
			StartLine:   msg.LineNo,
			EndLine:     msg.LineNo,
			LineType:    lineType,
			Body:        msg.Body,
			CodeSnippet: snippet,
		})
		m.focus = focusDiffViewer
		m.updateCommentMarkers()
		return m, nil

	case CommentCancelMsg:
		m.focus = focusDiffViewer
		return m, nil

	case finishMsg:
		m.output = comment.Format(m.comments.All())
		m.finished = true
		return m, tea.Quit

	case tea.KeyMsg:
		// Comment input gets priority when active
		if m.focus == focusCommentInput {
			var cmd tea.Cmd
			m.commentInput, cmd = m.commentInput.Update(msg)
			return m, cmd
		}

		// Search input gets priority when active
		if m.searching {
			switch msg.Type {
			case tea.KeyEscape:
				m.searching = false
				m.searchInput.Blur()
				return m, nil
			case tea.KeyEnter:
				term := m.searchInput.Value()
				m.searching = false
				m.searchInput.Blur()
				m.diffViewer.SetSearch(term)
				return m, nil
			}
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}

		return m.handleKeyMsg(msg)
	}

	return m, nil
}

func (m RootModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Help overlay dismissal
	if m.showHelp {
		if key == "?" || key == "esc" {
			m.showHelp = false
		}
		return m, nil
	}

	// ZZ key sequence
	if key == "Z" {
		if m.pendingZ {
			m.pendingZ = false
			m.output = comment.Format(m.comments.All())
			m.finished = true
			return m, tea.Quit
		}
		m.pendingZ = true
		return m, nil
	}
	m.pendingZ = false

	switch key {
	case "q":
		m.quitting = true
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "/":
		if m.focus == focusDiffViewer {
			m.searching = true
			m.searchInput.SetValue("")
			m.searchInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case "ctrl+d":
		if m.focus == focusDiffViewer {
			m.diffViewer, _ = m.diffViewer.Update(msg)
		}
		return m, nil

	case "ctrl+u":
		if m.focus == focusDiffViewer {
			m.diffViewer, _ = m.diffViewer.Update(msg)
		}
		return m, nil

	case "h":
		if m.focus == focusDiffViewer {
			m.focus = focusFileList
		}
		return m, nil

	case "l", "enter":
		if m.focus == focusFileList {
			m.focus = focusDiffViewer
			// Load diff for selected file
			sel := m.fileList.SelectedFile()
			if fd, err := m.git.FileDiff(m.base, sel.Path); err == nil {
				m.diffViewer.SetDiff(fd)
				m.updateCommentMarkers()
			}
		}
		return m, nil

	case "c":
		if m.focus == focusDiffViewer {
			line := m.diffViewer.CurrentLine()
			if line != nil {
				lineNo := m.diffViewer.CurrentLineNo()
				sel := m.fileList.SelectedFile()
				existing := ""
				if c := m.comments.Get(sel.Path, lineNo); c != nil {
					existing = c.Body
				}
				m.commentInput.Activate(sel.Path, lineNo, existing)
				m.focus = focusCommentInput
			}
		}
		return m, nil

	case "D":
		if m.focus == focusDiffViewer {
			lineNo := m.diffViewer.CurrentLineNo()
			sel := m.fileList.SelectedFile()
			m.comments.Delete(sel.Path, lineNo)
			m.updateCommentMarkers()
		}
		return m, nil
	}

	// Route to focused sub-model
	switch m.focus {
	case focusFileList:
		var cmd tea.Cmd
		m.fileList, cmd = m.fileList.Update(msg)
		// Auto-load diff when selection changes
		if key == "j" || key == "k" || key == "G" || key == "g" {
			sel := m.fileList.SelectedFile()
			if fd, err := m.git.FileDiff(m.base, sel.Path); err == nil {
				m.diffViewer.SetDiff(fd)
				m.updateCommentMarkers()
			}
		}
		return m, cmd

	case focusDiffViewer:
		var cmd tea.Cmd
		m.diffViewer, cmd = m.diffViewer.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *RootModel) updateCommentMarkers() {
	sel := m.fileList.SelectedFile()
	markers := make(map[int]bool)
	fileComments := m.comments.ForFile(sel.Path)
	if len(fileComments) > 0 {
		// Build a map of line numbers to flattened indices
		for i := 0; i < m.diffViewer.TotalLines(); i++ {
			dl := m.diffViewer.lineAt(i)
			if dl != nil && dl.line != nil {
				lineNo := dl.line.NewLineNo
				if dl.line.Type == git.LineRemoved {
					lineNo = dl.line.OldLineNo
				}
				for _, c := range fileComments {
					if lineNo == c.StartLine {
						markers[i] = true
					}
				}
			}
		}
	}
	m.diffViewer.SetCommentLines(markers)
}

// View renders the full UI.
func (m RootModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	if m.showHelp {
		return RenderHelp()
	}

	var b strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Render(fmt.Sprintf(" revui — %s → %s ", m.base, m.branch))
	b.WriteString(header)
	b.WriteString("\n")

	// File list panel
	fileListPanel := lipgloss.NewStyle().
		Width(m.fileListWidth).
		Height(m.height - 3).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Render(m.fileList.View())

	// Diff viewer panel
	diffPanel := lipgloss.NewStyle().
		Width(m.width - m.fileListWidth - 3).
		Height(m.height - 3).
		Render(m.diffViewer.View())

	// Main content
	content := lipgloss.JoinHorizontal(lipgloss.Top, fileListPanel, diffPanel)
	b.WriteString(content)
	b.WriteString("\n")

	// Status bar or overlay input
	if m.commentInput.Active() {
		b.WriteString(m.commentInput.View())
	} else if m.searching {
		searchBar := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("/") + m.searchInput.View()
		b.WriteString(searchBar)
	} else {
		b.WriteString(m.renderStatusBar())
	}

	return b.String()
}

func (m RootModel) renderStatusBar() string {
	commentCount := len(m.comments.All())
	status := fmt.Sprintf(" [c]omment  [v]isual  [Tab]view  [q]uit  [ZZ]done  [?]help  │  %d comments", commentCount)

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(status)
}

// Output returns the formatted comment output (available after finish).
func (m RootModel) Output() string {
	return m.output
}

// Finished returns whether the review was completed (not quit).
func (m RootModel) Finished() bool {
	return m.finished
}
