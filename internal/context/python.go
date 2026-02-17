package context

import (
	"os"
	"path/filepath"
)

// PythonParser detects Python project indicators and extracts project info.
type PythonParser struct{}

func (p *PythonParser) Name() string { return "python" }

func (p *PythonParser) Parse(repoRoot, cwd string) (*Result, error) {
	indicators := []string{
		"pyproject.toml",
		"setup.py",
		"setup.cfg",
		"requirements.txt",
		"Pipfile",
	}

	found := false
	for _, name := range indicators {
		if _, err := os.Stat(filepath.Join(repoRoot, name)); err == nil {
			found = true
			break
		}
	}
	if !found {
		return nil, nil
	}

	data := map[string]any{}

	// Check for pyproject.toml
	_, err := os.Stat(filepath.Join(repoRoot, "pyproject.toml"))
	data["has_pyproject"] = err == nil

	// Detect package manager from lockfiles
	data["package_manager"] = detectPythonPackageManager(repoRoot)

	// Detect virtual environment directory
	venvDirs := []string{".venv", "venv", ".env"}
	for _, vd := range venvDirs {
		info, err := os.Stat(filepath.Join(repoRoot, vd))
		if err == nil && info.IsDir() {
			data["venv"] = vd
			break
		}
	}

	return &Result{Name: p.Name(), Data: data}, nil
}

func detectPythonPackageManager(dir string) string {
	lockfiles := []struct {
		file string
		pm   string
	}{
		{"poetry.lock", "poetry"},
		{"uv.lock", "uv"},
		{"Pipfile.lock", "pipenv"},
		{"pdm.lock", "pdm"},
	}
	for _, lf := range lockfiles {
		if _, err := os.Stat(filepath.Join(dir, lf.file)); err == nil {
			return lf.pm
		}
	}
	return "pip"
}
