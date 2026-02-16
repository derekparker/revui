package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/deparker/revui/internal/git"
	"github.com/deparker/revui/internal/ui"
)

func main() {
	base := flag.String("base", "main", "base branch to diff against")
	flag.Parse()

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	runner := &git.Runner{Dir: dir}
	if !runner.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "Error: not a git repository")
		os.Exit(1)
	}

	if !runner.BranchExists(*base) {
		fmt.Fprintf(os.Stderr, "Error: base branch %q does not exist. Use --base to specify.\n", *base)
		os.Exit(1)
	}

	model := ui.NewRootModel(runner, *base, 80, 24)

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	rm := finalModel.(ui.RootModel)
	if rm.Finished() && rm.Output() != "" {
		if err := clipboard.WriteAll(rm.Output()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not copy to clipboard: %v\n", err)
			fmt.Fprintf(os.Stderr, "Printing to stdout instead:\n\n")
			fmt.Print(rm.Output())
		} else {
			fmt.Println("Review comments copied to clipboard.")
		}
	}
}
