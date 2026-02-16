package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes git commands in a working directory.
type Runner struct {
	Dir string
}

// CurrentBranch returns the name of the currently checked-out branch.
func (r *Runner) CurrentBranch() (string, error) {
	out, err := r.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// ChangedFiles returns the list of files changed between the given base ref and HEAD.
func (r *Runner) ChangedFiles(base string) ([]ChangedFile, error) {
	out, err := r.run("diff", "--name-status", base+"..HEAD")
	if err != nil {
		return nil, fmt.Errorf("getting changed files: %w", err)
	}
	return ParseNameStatus(out), nil
}

// FileDiff returns the parsed diff for a single file between the given base ref and HEAD.
func (r *Runner) FileDiff(base, path string) (*FileDiff, error) {
	out, err := r.run("diff", base+"..HEAD", "--", path)
	if err != nil {
		return nil, fmt.Errorf("getting diff for %s: %w", path, err)
	}
	diffs, err := ParseDiff(out)
	if err != nil {
		return nil, err
	}
	if len(diffs) == 0 {
		return &FileDiff{Path: path}, nil
	}
	diffs[0].Path = path
	return &diffs[0], nil
}

// IsGitRepo returns true if the working directory is inside a git repository.
func (r *Runner) IsGitRepo() bool {
	_, err := r.run("rev-parse", "--git-dir")
	return err == nil
}

// BranchExists returns true if the given branch name can be resolved.
func (r *Runner) BranchExists(branch string) bool {
	_, err := r.run("rev-parse", "--verify", branch)
	return err == nil
}

func (r *Runner) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return "", err
	}
	return string(out), nil
}
