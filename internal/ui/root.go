package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/comment"
	"github.com/deparker/revui/internal/git"
	"github.com/deparker/revui/internal/output"
)

type focusArea int

const (
	focusFileList focusArea = iota
	focusDiffViewer
	focusCommentInput
	focusOutputSelect
)

type reviewMode int

const (
	modeBranch reviewMode = iota
	modeUncommitted
)

// GitRunner is the interface for git operations, enabling testing with mocks.
type GitRunner interface {
	ChangedFiles(base string) ([]git.ChangedFile, error)
	FileDiff(base, path string) (*git.FileDiff, error)
	CurrentBranch() (string, error)
	HasUncommittedChanges() bool
	UncommittedFiles() ([]git.ChangedFile, error)
	UncommittedFileDiff(path string) (*git.FileDiff, error)
}

// finishMsg signals the review is done and comments should be copied.
type finishMsg struct{}

// tickRefreshMsg signals that it's time to check for uncommitted changes.
type tickRefreshMsg struct{}

// refreshResultMsg carries the results of an async refresh operation.
type refreshResultMsg struct {
	files         []git.ChangedFile
	diff          *git.FileDiff
	requestedPath string // the file path that was selected when the refresh started
	err           error
}

// RootModel is the top-level Bubble Tea model.
type RootModel struct {
	git           GitRunner
	mode          reviewMode
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
	showHelp          bool
	searchInput       textinput.Model
	searching         bool
	refreshInProgress bool
	outputSelector    OutputSelector
	deliveryResult    string // status message after delivery
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

// NewRootModelUncommitted creates the root model for reviewing uncommitted changes.
func NewRootModelUncommitted(gitRunner GitRunner, width, height int) RootModel {
	fileListWidth := 30

	files, err := gitRunner.UncommittedFiles()
	if err != nil {
		return RootModel{err: err}
	}

	fl := NewFileList(files, fileListWidth, height-2)
	dv := NewDiffViewer(width-fileListWidth-3, height-2)
	ci := NewCommentInput(width)

	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 100
	si.Width = width - 10

	// Load the first file's diff if available
	if len(files) > 0 {
		if fd, err := gitRunner.UncommittedFileDiff(files[0].Path); err == nil {
			dv.SetDiff(fd)
		}
	}

	return RootModel{
		git:           gitRunner,
		mode:          modeUncommitted,
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
	if m.mode == modeUncommitted {
		return scheduleRefreshTick()
	}
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

	case tickRefreshMsg:
		if m.mode != modeUncommitted {
			return m, nil
		}
		if m.refreshInProgress {
			return m, scheduleRefreshTick()
		}
		m.refreshInProgress = true
		return m, m.refreshCmd()

	case refreshResultMsg:
		m.refreshInProgress = false
		if msg.err != nil {
			return m, scheduleRefreshTick()
		}

		// Update file list
		m.files = msg.files
		m.fileList.SetFiles(msg.files)

		// Update diff only if the user is still on the same file
		currentPath := ""
		if len(m.files) > 0 {
			currentPath = m.fileList.SelectedFile().Path
		}
		if msg.diff != nil && msg.requestedPath == currentPath {
			m.diffViewer.RefreshDiff(msg.diff)
			m.updateCommentMarkers()
		} else if currentPath == "" {
			// All files removed
			m.diffViewer.RefreshDiff(nil)
		}

		return m, scheduleRefreshTick()

	case CommentSubmitMsg:
		m.comments.Add(comment.Comment{
			FilePath:  msg.FilePath,
			StartLine: msg.LineNo,
			EndLine:   msg.EndLineNo,
			LineType:  msg.LineType,
			Body:      msg.Body,
		})
		m.focus = focusDiffViewer
		m.updateCommentMarkers()
		return m, nil

	case CommentCancelMsg:
		m.focus = focusDiffViewer
		return m, nil

	case OutputSelectMsg:
		result, err := output.Deliver(msg.Target, m.output)
		if err != nil {
			m.outputSelector.SetError(err.Error())
			return m, nil
		}
		m.deliveryResult = result
		m.finished = true
		return m, tea.Quit

	case OutputCancelMsg:
		m.quitting = true
		return m, tea.Quit

	case navigateFileMsg:
		var switched bool
		if msg.direction > 0 {
			switched = m.fileList.SelectNext()
		} else {
			switched = m.fileList.SelectPrev()
		}
		if switched {
			sel := m.fileList.SelectedFile()
			if fd, err := m.loadFileDiff(sel.Path); err == nil {
				m.diffViewer.SetDiff(fd)
				if msg.direction < 0 {
					m.diffViewer.SetCursorToEnd()
				}
				m.updateCommentMarkers()
			}
		}
		return m, nil

	case finishMsg:
		m.output = comment.Format(m.comments.All())
		if m.output == "" {
			m.finished = true
			return m, tea.Quit
		}
		targets := output.DetectTargets(os.Getenv("TMUX"), os.Getenv("TMUX_PANE"))
		m.outputSelector = NewOutputSelector(targets, m.width, m.height)
		m.focus = focusOutputSelect
		return m, nil

	case tea.KeyMsg:
		// Comment input gets priority when active
		if m.focus == focusCommentInput {
			var cmd tea.Cmd
			m.commentInput, cmd = m.commentInput.Update(msg)
			return m, cmd
		}

		// Output selector gets priority when active
		if m.focus == focusOutputSelect {
			var cmd tea.Cmd
			m.outputSelector, cmd = m.outputSelector.Update(msg)
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
			if m.output == "" {
				// No comments — quit directly
				m.finished = true
				return m, tea.Quit
			}
			// Show output selector
			targets := output.DetectTargets(os.Getenv("TMUX"), os.Getenv("TMUX_PANE"))
			m.outputSelector = NewOutputSelector(targets, m.width, m.height)
			m.focus = focusOutputSelect
			return m, nil
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
			if fd, err := m.loadFileDiff(sel.Path); err == nil {
				m.diffViewer.SetDiff(fd)
				m.updateCommentMarkers()
			}
		}
		return m, nil

	case "c":
		if m.focus == focusDiffViewer {
			sel := m.fileList.SelectedFile()

			if m.diffViewer.InVisualMode() {
				// Range comment from visual selection
				vStart, vEnd := m.diffViewer.VisualRange()
				startLineNo := m.diffViewer.LineNoAt(vStart)
				endLineNo := m.diffViewer.LineNoAt(vEnd)
				lineType := git.LineContext
				if dl := m.diffViewer.lineAt(vStart); dl != nil && dl.line != nil {
					lineType = dl.line.Type
				}
				m.diffViewer.ExitVisualMode()
				m.commentInput.Activate(sel.Path, startLineNo, endLineNo, lineType, "")
				m.focus = focusCommentInput
			} else {
				// Single-line comment
				line := m.diffViewer.CurrentLine()
				if line != nil {
					lineNo := m.diffViewer.CurrentLineNo()
					existing := ""
					if c := m.comments.Get(sel.Path, lineNo); c != nil {
						existing = c.Body
					}
					m.commentInput.Activate(sel.Path, lineNo, lineNo, line.Type, existing)
					m.focus = focusCommentInput
				} else if sel.Status == "B" {
					// Binary file: allow comment on file itself
					existing := ""
					if c := m.comments.Get(sel.Path, 0); c != nil {
						existing = c.Body
					}
					m.commentInput.Activate(sel.Path, 0, 0, git.LineContext, existing)
					m.focus = focusCommentInput
				}
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
			if fd, err := m.loadFileDiff(sel.Path); err == nil {
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

// loadFileDiff loads the diff for the given path based on the current review mode.
func (m *RootModel) loadFileDiff(path string) (*git.FileDiff, error) {
	if m.mode == modeUncommitted {
		return m.git.UncommittedFileDiff(path)
	}
	return m.git.FileDiff(m.base, path)
}

const refreshInterval = 2 * time.Second

// scheduleRefreshTick returns a tea.Cmd that sends a tickRefreshMsg after the refresh interval.
func scheduleRefreshTick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickRefreshMsg{}
	})
}

// refreshCmd returns a tea.Cmd that asynchronously fetches the current file list
// and diff for the selected file.
func (m RootModel) refreshCmd() tea.Cmd {
	currentPath := ""
	if len(m.files) > 0 {
		currentPath = m.fileList.SelectedFile().Path
	}
	gitRunner := m.git

	return func() tea.Msg {
		files, err := gitRunner.UncommittedFiles()
		if err != nil {
			return refreshResultMsg{err: err}
		}

		var diff *git.FileDiff
		if currentPath != "" {
			for _, f := range files {
				if f.Path == currentPath {
					diff, _ = gitRunner.UncommittedFileDiff(currentPath)
					break
				}
			}
		}

		return refreshResultMsg{
			files:         files,
			diff:          diff,
			requestedPath: currentPath,
		}
	}
}

// View renders the full UI.
func (m RootModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	if m.showHelp {
		return RenderHelp()
	}

	if m.focus == focusOutputSelect {
		return m.outputSelector.View()
	}

	var b strings.Builder

	// Header
	var headerText string
	if m.mode == modeUncommitted {
		headerText = " revui — uncommitted changes "
	} else {
		headerText = fmt.Sprintf(" revui — %s → %s ", m.base, m.branch)
	}
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Render(headerText)
	b.WriteString(header)
	b.WriteString("\n")

	// Set focus state for sub-models
	m.fileList.focused = m.focus == focusFileList
	m.diffViewer.focused = m.focus == focusDiffViewer

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

// DeliveryResult returns the status message from delivery (available after finish).
func (m RootModel) DeliveryResult() string {
	return m.deliveryResult
}
