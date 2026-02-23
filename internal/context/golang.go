package context

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
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

	// Parse module name from first "module " line.
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			data["module"] = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}

	// Walk the tree to find Go packages and which have tests.
	packages, testPackages := findGoPackages(repoRoot)
	if len(packages) > 0 {
		data["packages"] = packages
	}
	if len(testPackages) > 0 {
		data["test_packages"] = testPackages
	}

	return &Result{Name: g.Name(), Data: data}, nil
}

// skipDirs are directory names that should never be walked.
var skipDirs = map[string]bool{
	"vendor": true, "node_modules": true, "testdata": true,
	".git": true, ".hg": true, ".svn": true,
}

// findGoPackages walks the repo tree and returns all Go package paths
// (relative, with "./" prefix) and the subset that contain test files.
func findGoPackages(root string) (packages, testPackages []string) {
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}
		return nil
	})

	// Second pass: for each directory, check for .go and _test.go files.
	seen := map[string]bool{}
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		dir := filepath.Dir(path)
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return nil
		}
		pkg := "./" + filepath.ToSlash(rel)
		if pkg == "./." {
			pkg = "."
		}

		if !seen[pkg] {
			seen[pkg] = true
			packages = append(packages, pkg)
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			seen[pkg+"__test"] = true
		}
		return nil
	})

	sort.Strings(packages)

	for _, p := range packages {
		if seen[p+"__test"] {
			testPackages = append(testPackages, p)
		}
	}

	return packages, testPackages
}
