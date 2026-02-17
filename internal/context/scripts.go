package context

import (
	"os"
	"path/filepath"
	"sort"
)

// ScriptsParser looks for bin/ and scripts/ directories and lists script files.
type ScriptsParser struct{}

func (s *ScriptsParser) Name() string { return "scripts" }

func (s *ScriptsParser) Parse(repoRoot, cwd string) (*Result, error) {
	var scripts []string

	for _, dirName := range []string{"bin", "scripts"} {
		dirPath := filepath.Join(repoRoot, dirName)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue // directory doesn't exist or can't be read
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			scripts = append(scripts, filepath.Join(dirName, e.Name()))
		}
	}

	if len(scripts) == 0 {
		return nil, nil
	}

	sort.Strings(scripts)

	return &Result{
		Name: s.Name(),
		Data: map[string]any{
			"scripts": scripts,
		},
	}, nil
}
