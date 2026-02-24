package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// findFileUpward walks from start up to root looking for filename.
// Returns the full path if found, or empty string if not.
func findFileUpward(filename, start, root string) string {
	start = filepath.Clean(start)
	root = filepath.Clean(root)
	dir := start
	for {
		candidate := filepath.Join(dir, filename)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		if dir == root {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// NodeParser detects package.json and extracts Node.js project info.
type NodeParser struct{}

func (n *NodeParser) Name() string { return "node" }

func (n *NodeParser) Parse(repoRoot, cwd string) (*Result, error) {
	pkgPath := findFileUpward("package.json", cwd, repoRoot)
	if pkgPath == "" {
		return nil, nil
	}

	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Scripts    map[string]string `json:"scripts"`
		Workspaces []string          `json:"workspaces"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, err
	}

	data := map[string]any{}

	if len(pkg.Scripts) > 0 {
		scripts := make([]string, 0, len(pkg.Scripts))
		for k := range pkg.Scripts {
			scripts = append(scripts, k)
		}
		sort.Strings(scripts)
		data["scripts"] = scripts
	}

	if len(pkg.Workspaces) > 0 {
		data["workspaces"] = pkg.Workspaces
	}

	// Detect package manager from lockfiles in the same directory as package.json
	pkgDir := filepath.Dir(pkgPath)
	data["package_manager"] = detectPackageManager(pkgDir)

	return &Result{Name: n.Name(), Data: data}, nil
}

func detectPackageManager(dir string) string {
	lockfiles := []struct {
		file string
		pm   string
	}{
		{"bun.lockb", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	}
	for _, lf := range lockfiles {
		if _, err := os.Stat(filepath.Join(dir, lf.file)); err == nil {
			return lf.pm
		}
	}
	return "npm"
}
