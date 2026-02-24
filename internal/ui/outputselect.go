package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/deparker/revui/internal/output"
)

// OutputSelectMsg is sent when the user selects an output target.
type OutputSelectMsg struct {
	Target output.OutputTarget
}

// OutputCancelMsg is sent when the user cancels the output selection.
type OutputCancelMsg struct{}

// OutputSelector is a sub-model for selecting an output target.
type OutputSelector struct {
	targets []output.OutputTarget
	cursor  int
	width   int
	height  int
	err     string // delivery error to display
}

// NewOutputSelector creates a new output selector component.
func NewOutputSelector(targets []output.OutputTarget, width, height int) OutputSelector {
	return OutputSelector{
		targets: targets,
		cursor:  0,
		width:   width,
		height:  height,
	}
}

// SetError sets an error message to display (called when delivery fails).
func (os *OutputSelector) SetError(msg string) {
	os.err = msg
}

// Update handles key messages.
func (os OutputSelector) Update(msg tea.Msg) (OutputSelector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			switch msg.String() {
			case "j":
				if os.cursor < len(os.targets)-1 {
					os.cursor++
				}
			case "k":
				if os.cursor > 0 {
					os.cursor--
				}
			case "q":
				return os, func() tea.Msg { return OutputCancelMsg{} }
			}
		case tea.KeyDown:
			if os.cursor < len(os.targets)-1 {
				os.cursor++
			}
		case tea.KeyUp:
			if os.cursor > 0 {
				os.cursor--
			}
		case tea.KeyEnter:
			if len(os.targets) > 0 {
				return os, func() tea.Msg {
					return OutputSelectMsg{Target: os.targets[os.cursor]}
				}
			}
		case tea.KeyEscape:
			return os, func() tea.Msg { return OutputCancelMsg{} }
		}
	}

	return os, nil
}

// View renders the selection list.
func (os OutputSelector) View() string {
	if len(os.targets) == 0 {
		return renderEmptyView()
	}

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	normalStyle := lipgloss.NewStyle()
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	var s strings.Builder
	s.WriteString(titleStyle.Render("Send review to:"))
	s.WriteString("\n")

	// Find the index where Claude targets end (to insert separator)
	claudeEndIdx := -1
	for i, target := range os.targets {
		if target.Kind != output.TargetClaude {
			claudeEndIdx = i
			break
		}
	}

	for i, target := range os.targets {
		// Insert separator between Claude targets and fallback targets
		if i == claudeEndIdx && claudeEndIdx > 0 {
			s.WriteString(separatorStyle.Render("  ── or ──"))
			s.WriteString("\n")
		}

		var line string
		if i == os.cursor {
			line = selectedStyle.Render("  > " + target.Label)
		} else {
			line = normalStyle.Render("    " + target.Label)
		}
		s.WriteString(line)
		s.WriteString("\n")
	}

	// Show error if present
	if os.err != "" {
		s.WriteString("\n")
		s.WriteString(errorStyle.Render("  Error: " + os.err))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(footerStyle.Render("  [Enter] select  [q] cancel"))

	return s.String()
}

// renderEmptyView renders the view when no targets are available.
func renderEmptyView() string {
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	normalStyle := lipgloss.NewStyle()
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var s strings.Builder
	s.WriteString(titleStyle.Render("Send review to:"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("  No output targets available."))
	s.WriteString("\n\n")
	s.WriteString(footerStyle.Render("  [q] cancel"))

	return s.String()
}

// SetSize updates the dimensions.
func (os *OutputSelector) SetSize(width, height int) {
	os.width = width
	os.height = height
}
