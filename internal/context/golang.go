package context

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// GoParser detects go.mod in the repo root and extracts Go project info.
type GoParser struct{}

func (g *GoParser) Name() string { return "go" }

func (g *GoParser) Parse(repoRoot, cwd string) (*Result, error) {
	goModPath := filepath.Join(repoRoot, "go.mod")
	f, err := os.Open(goModPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data := map[string]any{}

	// Parse module name from first "module " line
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			data["module"] = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}

	// Check for _test.go files in root + one level of subdirs
	data["has_tests"] = hasGoTests(repoRoot)

	return &Result{Name: g.Name(), Data: data}, nil
}

func hasGoTests(root string) bool {
	// Check root directory
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}

	// Check one level of subdirectories
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Skip hidden directories and common non-source dirs
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		subEntries, err := os.ReadDir(filepath.Join(root, e.Name()))
		if err != nil {
			continue
		}
		for _, se := range subEntries {
			if !se.IsDir() && strings.HasSuffix(se.Name(), "_test.go") {
				return true
			}
		}
	}

	return false
}
