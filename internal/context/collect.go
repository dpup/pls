package context

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultParsers returns all v0 parsers.
func DefaultParsers() []Parser {
	return []Parser{
		&GitParser{},
		&NodeParser{},
		&MakeParser{},
		&JustParser{},
		&GoParser{},
		&DockerParser{},
		&PythonParser{},
		&ScriptsParser{},
	}
}

// Collect runs all parsers and builds a Snapshot.
func Collect(cwd string, parsers []Parser) (*Snapshot, error) {
	repoRoot := findRepoRoot(cwd)
	if repoRoot == "" {
		repoRoot = cwd
	}

	cwdRel, err := filepath.Rel(repoRoot, cwd)
	if err != nil {
		cwdRel = "."
	}

	snap := &Snapshot{
		RepoRoot: repoRoot,
		CwdRel:   cwdRel,
	}

	for _, p := range parsers {
		result, err := p.Parse(repoRoot, cwd)
		if err != nil {
			continue
		}
		if result != nil {
			snap.Results = append(snap.Results, *result)
		}
	}

	return snap, nil
}

func findRepoRoot(cwd string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
