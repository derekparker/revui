package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/deparker/revui/internal/git"
	"github.com/deparker/revui/internal/ui"
)

func main() {
	base := flag.String("base", "", "base branch to diff against (auto-detected if not set)")
	remote := flag.String("remote", "origin", "remote to detect default branch from")
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

	var model ui.RootModel
	if runner.HasUncommittedChanges() {
		model = ui.NewRootModelUncommitted(runner, 80, 24)
	} else {
		// Auto-detect base branch if not explicitly provided
		baseBranch := *base
		if baseBranch == "" {
			baseBranch = runner.DefaultBranch(*remote)
		}

		if !runner.BranchExists(baseBranch) {
			fmt.Fprintf(os.Stderr, "Error: base branch %q does not exist. Use --base to specify.\n", baseBranch)
			os.Exit(1)
		}

		model = ui.NewRootModel(runner, baseBranch, 80, 24)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	rm, ok := finalModel.(ui.RootModel)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unexpected model type\n")
		os.Exit(1)
	}
	if rm.Finished() && rm.DeliveryResult() != "" {
		fmt.Println(rm.DeliveryResult())
	}
}
