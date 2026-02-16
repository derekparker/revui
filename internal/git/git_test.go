package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "checkout", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", args, err, out)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n\nfunc hello() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "initial")
	runCmd(t, dir, "git", "checkout", "-b", "feature")

	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n\nfunc hello() {\n\tfmt.Println(\"hello\")\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "world.go"), []byte("package main\n\nfunc world() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "add feature")

	return dir
}

func runCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v failed: %s\n%s", args, err, out)
	}
}

func TestChangedFiles(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}
	files, err := r.ChangedFiles("main")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 changed files, got %d", len(files))
	}
	paths := map[string]string{}
	for _, f := range files {
		paths[f.Path] = f.Status
	}
	if paths["hello.go"] != "M" {
		t.Errorf("hello.go status = %q, want M", paths["hello.go"])
	}
	if paths["world.go"] != "A" {
		t.Errorf("world.go status = %q, want A", paths["world.go"])
	}
}

func TestFileDiff(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}
	fd, err := r.FileDiff("main", "hello.go")
	if err != nil {
		t.Fatal(err)
	}
	if fd.Path != "hello.go" {
		t.Errorf("path = %q, want %q", fd.Path, "hello.go")
	}
	if len(fd.Hunks) == 0 {
		t.Fatal("expected at least one hunk")
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}
	branch, err := r.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feature" {
		t.Errorf("branch = %q, want %q", branch, "feature")
	}
}

func TestIsGitRepo(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}
	if !r.IsGitRepo() {
		t.Error("expected IsGitRepo to return true for a git repo")
	}

	notRepo := t.TempDir()
	r2 := &Runner{Dir: notRepo}
	if r2.IsGitRepo() {
		t.Error("expected IsGitRepo to return false for a non-repo directory")
	}
}

func TestBranchExists(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}
	if !r.BranchExists("main") {
		t.Error("expected BranchExists to return true for 'main'")
	}
	if r.BranchExists("nonexistent-branch") {
		t.Error("expected BranchExists to return false for 'nonexistent-branch'")
	}
}

func TestDefaultBranch(t *testing.T) {
	dir := setupTestRepo(t)
	r := &Runner{Dir: dir}

	// No remote configured, so should fall back to "main"
	branch := r.DefaultBranch("origin")
	if branch != "main" {
		t.Errorf("DefaultBranch with no remote = %q, want %q", branch, "main")
	}
}

func TestDefaultBranchWithRemote(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a bare remote and set its HEAD
	remoteDir := t.TempDir()
	runCmd(t, remoteDir, "git", "init", "--bare")
	runCmd(t, dir, "git", "remote", "add", "origin", remoteDir)
	runCmd(t, dir, "git", "push", "origin", "main")
	runCmd(t, dir, "git", "remote", "set-head", "origin", "main")

	r := &Runner{Dir: dir}
	branch := r.DefaultBranch("origin")
	if branch != "main" {
		t.Errorf("DefaultBranch = %q, want %q", branch, "main")
	}
}
