package context

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var justRecipeRe = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*)\s*.*:`)

// JustParser detects a Justfile in the repo root and extracts recipes.
type JustParser struct{}

func (j *JustParser) Name() string { return "just" }

func (j *JustParser) Parse(repoRoot, cwd string) (*Result, error) {
	var raw []byte
	var err error

	// Check for Justfile, justfile, .justfile (in order)
	for _, name := range []string{"Justfile", "justfile", ".justfile"} {
		raw, err = os.ReadFile(filepath.Join(repoRoot, name))
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if raw == nil {
		return nil, nil
	}

	var recipes []string
	for _, line := range strings.Split(string(raw), "\n") {
		// Skip indented lines and comments
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if m := justRecipeRe.FindStringSubmatch(line); m != nil {
			recipes = append(recipes, m[1])
		}
	}

	if len(recipes) == 0 {
		return nil, nil
	}

	return &Result{
		Name: j.Name(),
		Data: map[string]any{
			"recipes": recipes,
		},
	}, nil
}
