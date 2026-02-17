package context

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var makeTargetRe = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*)\s*:`)

// MakeParser detects a Makefile in the repo root and extracts targets.
type MakeParser struct{}

func (m *MakeParser) Name() string { return "make" }

func (m *MakeParser) Parse(repoRoot, cwd string) (*Result, error) {
	path := filepath.Join(repoRoot, "Makefile")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var targets []string
	for _, line := range strings.Split(string(raw), "\n") {
		// Skip indented lines (tab-prefixed) and comments
		if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "#") {
			continue
		}
		if m := makeTargetRe.FindStringSubmatch(line); m != nil {
			targets = append(targets, m[1])
		}
	}

	if len(targets) == 0 {
		return nil, nil
	}

	return &Result{
		Name: m.Name(),
		Data: map[string]any{
			"targets": targets,
		},
	}, nil
}
