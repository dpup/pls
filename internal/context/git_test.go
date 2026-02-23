package context

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitParser_InRepo(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	writeFile(t, filepath.Join(dir, "file.txt"), "hello")
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "init")

	p := &GitParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}
	if result.Data["repo_root"] != dir {
		t.Errorf("expected repo_root %q, got %q", dir, result.Data["repo_root"])
	}
	if _, ok := result.Data["branch"]; !ok {
		t.Error("expected branch in result")
	}
}

func TestGitParser_NotARepo(t *testing.T) {
	dir := t.TempDir()
	p := &GitParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for non-repo dir")
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
