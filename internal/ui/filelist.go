package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/git"
)

var (
	selectedStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	unselectedStyle     = lipgloss.NewStyle()
	statusAddedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	statusModifiedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	statusDeletedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

// FileList is a Bubble Tea sub-model for displaying changed files.
type FileList struct {
	files   []git.ChangedFile
	cursor  int
	width   int
	height  int
	focused bool
}

// NewFileList creates a new file list with the given changed files.
func NewFileList(files []git.ChangedFile, width, height int) FileList {
	return FileList{
		files:  files,
		cursor: 0,
		width:  width,
		height: height,
	}
}

// Init returns no initial command.
func (fl FileList) Init() tea.Cmd {
	return nil
}

// Update handles key messages for vim-style navigation.
func (fl FileList) Update(msg tea.Msg) (FileList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if fl.cursor < len(fl.files)-1 {
				fl.cursor++
			}
		case "k", "up":
			if fl.cursor > 0 {
				fl.cursor--
			}
		case "G":
			fl.cursor = len(fl.files) - 1
		case "g":
			// gg handled by root model tracking "g" prefix
			fl.cursor = 0
		}
	}
	return fl, nil
}

// View renders the file list.
func (fl FileList) View() string {
	if len(fl.files) == 0 {
		return "No changed files"
	}

	var b strings.Builder
	for i, f := range fl.files {
		icon := statusIcon(f.Status)
		line := icon + " " + f.Path

		if i == fl.cursor {
			line = selectedStyle.Render("▸ " + line)
		} else {
			line = unselectedStyle.Render("  " + line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// SelectedFile returns the currently selected file.
func (fl FileList) SelectedFile() git.ChangedFile {
	if fl.cursor >= 0 && fl.cursor < len(fl.files) {
		return fl.files[fl.cursor]
	}
	return git.ChangedFile{}
}

// SelectedIndex returns the current cursor index.
func (fl FileList) SelectedIndex() int {
	return fl.cursor
}

// SelectNext moves the cursor to the next file. Returns false if already at the last file.
func (fl *FileList) SelectNext() bool {
	if fl.cursor >= len(fl.files)-1 {
		return false
	}
	fl.cursor++
	return true
}

// SelectPrev moves the cursor to the previous file. Returns false if already at the first file.
func (fl *FileList) SelectPrev() bool {
	if fl.cursor <= 0 {
		return false
	}
	fl.cursor--
	return true
}

// SetFocused sets whether this component has focus.
func (fl *FileList) SetFocused(focused bool) {
	fl.focused = focused
}

// SetSize updates the dimensions.
func (fl *FileList) SetSize(width, height int) {
	fl.width = width
	fl.height = height
}

func statusIcon(status string) string {
	switch status {
	case "A":
		return statusAddedStyle.Render("A")
	case "M":
		return statusModifiedStyle.Render("M")
	case "D":
		return statusDeletedStyle.Render("D")
	case "R":
		return statusModifiedStyle.Render("R")
	default:
		return "?"
	}
}
