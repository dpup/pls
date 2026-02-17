package context

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitParser struct{}

func (g *GitParser) Name() string { return "git" }

func (g *GitParser) Parse(repoRoot, cwd string) (*Result, error) {
	if _, err := os.Stat(filepath.Join(repoRoot, ".git")); os.IsNotExist(err) {
		return nil, nil
	}

	data := map[string]any{
		"repo_root": repoRoot,
	}

	if branch, err := gitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		data["branch"] = branch
	}

	if changed, err := gitOutput(repoRoot, "diff", "--name-only", "HEAD"); err == nil && changed != "" {
		data["changed_files"] = strings.Split(changed, "\n")
	}

	return &Result{Name: g.Name(), Data: data}, nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
