package ui

import "github.com/charmbracelet/lipgloss"

var helpStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("12")).
	Padding(1, 2)

// RenderHelp returns the help overlay text.
func RenderHelp() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("revui — Keybindings")

	help := title + "\n\n" +
		"Navigation\n" +
		"  j/k         Move down/up\n" +
		"  h/l         Switch panel (file list ↔ diff)\n" +
		"  G           Jump to bottom\n" +
		"  gg          Jump to top\n" +
		"  Ctrl+d/u    Half-page down/up\n" +
		"  Ctrl+f/b    Full-page down/up\n" +
		"  [/]         Jump to prev/next change\n" +
		"  {/}         Jump to prev/next hunk\n" +
		"\n" +
		"Commenting\n" +
		"  c           Add/edit comment on current line\n" +
		"  D           Delete comment on current line\n" +
		"  v           Visual mode (select line range)\n" +
		"  ]c/[c       Jump to next/prev comment\n" +
		"\n" +
		"Views\n" +
		"  Tab         Toggle unified/side-by-side view\n" +
		"  /           Search in diff\n" +
		"  n/N         Next/prev search result\n" +
		"\n" +
		"Actions\n" +
		"  ZZ          Finish review (copy to clipboard)\n" +
		"  q           Quit without copying\n" +
		"  ?           Toggle this help\n" +
		"\n" +
		"Press ? or Esc to close"

	return helpStyle.Render(help)
}
