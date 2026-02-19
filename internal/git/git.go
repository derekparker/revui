package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// DefaultBranch returns the default branch for the given remote by reading
// the symbolic ref. Falls back to "main" if detection fails.
func (r *Runner) DefaultBranch(remote string) string {
	out, err := r.run("symbolic-ref", "refs/remotes/"+remote+"/HEAD")
	if err != nil {
		return "main"
	}
	// Output is like "refs/remotes/origin/main\n"
	ref := strings.TrimSpace(out)
	prefix := "refs/remotes/" + remote + "/"
	if after, ok := strings.CutPrefix(ref, prefix); ok {
		return after
	}
	return "main"
}

// HasUncommittedChanges returns true if there are staged, unstaged, or untracked changes.
func (r *Runner) HasUncommittedChanges() bool {
	out, err := r.run("status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// UncommittedFiles returns changed files (staged + unstaged vs HEAD) plus untracked files.
// Binary files are marked with status "B".
func (r *Runner) UncommittedFiles() ([]ChangedFile, error) {
	// Get tracked changes (staged + unstaged)
	diffOut, err := r.run("diff", "HEAD", "--name-status")
	if err != nil {
		// If HEAD doesn't exist (initial commit), try --cached
		diffOut, err = r.run("diff", "--cached", "--name-status")
		if err != nil {
			diffOut = ""
		}
	}
	files := ParseNameStatus(diffOut)

	// Identify binary files among tracked changes via --numstat
	binaries := r.detectBinaryTracked()

	// Mark binary tracked files
	for i := range files {
		if binaries[files[i].Path] {
			files[i].Status = "B"
		}
	}

	// Get untracked files
	untrackedOut, err := r.run("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return files, nil
	}

	seen := make(map[string]bool, len(files))
	for _, f := range files {
		seen[f.Path] = true
	}

	for line := range strings.SplitSeq(strings.TrimSpace(untrackedOut), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		status := "A"
		if r.isBinaryFile(line) {
			status = "B"
		}
		files = append(files, ChangedFile{Path: line, Status: status})
	}

	return files, nil
}

// detectBinaryTracked returns a set of paths that are binary among tracked changes.
func (r *Runner) detectBinaryTracked() map[string]bool {
	out, err := r.run("diff", "HEAD", "--numstat")
	if err != nil {
		return nil
	}
	binaries := make(map[string]bool)
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Binary files show as "-\t-\tfilename"
		if strings.HasPrefix(line, "-\t-\t") {
			path := line[4:]
			binaries[path] = true
		}
	}
	return binaries
}

// isBinaryFile checks if a file appears to be binary by looking for null bytes in the first 8KB.
func (r *Runner) isBinaryFile(path string) bool {
	fullPath := filepath.Join(r.Dir, path)
	f, err := os.Open(fullPath)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 8192)
	n, err := f.Read(buf)
	if n == 0 {
		return false
	}
	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}
	return false
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
